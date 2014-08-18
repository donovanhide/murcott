package murcott

import (
	"github.com/vmihailenco/msgpack"
	"net"
)

const (
	portBegin = 9200
	portEnd   = 9220
	bootstrap = "bt.murcott.net"
)

type nodeInfo struct {
	Id   NodeId
	Addr *net.UDPAddr
}

type message struct {
	id      NodeId
	payload []byte
}

type incomingPacket struct {
	packet packet
	addr   *net.UDPAddr
}

type node struct {
	selfnode       nodeInfo
	dht            *dht
	conn           *net.UDPConn
	key            *PrivateKey
	keycache       map[string]PublicKey
	waitingPackets []incomingPacket
	logger         *Logger
	msgcb          []func(NodeId, []byte)
	msgChan        chan message
	sendChan       chan packet
	exit           chan struct{}
}

type udpDatagram struct {
	Data []byte
	Addr *net.UDPAddr
}

func getOpenPortConn() (*net.UDPConn, int) {
	for port := portBegin; port <= portEnd; port++ {
		addr := net.UDPAddr{
			Port: port,
			IP:   net.ParseIP("127.0.0.1"),
		}
		conn, err := net.ListenUDP("udp", &addr)
		if err == nil {
			return conn, port
		}
	}
	return nil, 0
}

func newNode(key *PrivateKey, logger *Logger) *node {
	selfnode := nodeInfo{Id: key.PublicKeyHash()}
	dht := newDht(10, selfnode, logger)
	exit := make(chan struct{})

	node := node{
		selfnode: selfnode,
		conn:     nil,
		key:      key,
		keycache: make(map[string]PublicKey),
		dht:      dht,
		logger:   logger,
		msgChan:  make(chan message, 100),
		sendChan: make(chan packet, 100),
		exit:     exit,
	}

	return &node
}

func (p *node) run() {

	conn, selfport := getOpenPortConn()
	p.conn = conn

	p.logger.Info("Node ID: %s", p.selfnode.Id.String())
	p.logger.Info("Node UDP port: %d", selfport)

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	// portscan
	for port := portBegin; port <= portEnd; port++ {
		if port != selfport {
			p.sendDiscoveryPacket(&net.UDPAddr{Port: port, IP: host[0]})
		}
	}

	p.logger.Info("Sent discovery packet to %v:%d-%d", host[0], portBegin, portEnd)

	datach := make(chan udpDatagram)
	rpcch := p.dht.rpcChannel()

	// read datagram from udp socket
	go func() {
		for {
			var buf [1024]byte
			len, addr, err := conn.ReadFromUDP(buf[:])
			if err != nil {
				break
			}
			datach <- udpDatagram{buf[:len], addr}
		}
	}()

	go p.dht.run()

	for {
		select {

		case packet := <-p.sendChan:
			packet.sign(p.key)
			data, err := msgpack.Marshal(packet)
			if err == nil {
				node := p.dht.getNodeInfo(packet.Dst)
				if node != nil {
					p.conn.WriteToUDP(data, node.Addr)
				} else {
					p.logger.Error("route not found: %s", packet.Dst.String())
				}
			} else {
				p.logger.Error("packet marshal error")
			}

		case data := <-datach:
			var packet packet
			err = msgpack.Unmarshal(data.Data, &packet)
			if err != nil {
				continue
			}
			p.logger.Info("Receive %s packet from %s", packet.Type, packet.Src.String())

			if packet.Type == "key" {
				if len(packet.Payload) == 0 {
					p.sendPublicKeyResponse(data.Addr)
				} else {
					p.processPublicKeyResponse(packet)
				}
			} else {
				// find publickey from cache
				if key, ok := p.keycache[packet.Src.String()]; ok {
					if packet.verify(&key) {
						p.processPacket(packet, data.Addr)
					}
				} else {
					// request publickey
					p.sendPacketAddr(data.Addr, "key", nil)
					p.addWaitingPacket(incomingPacket{packet: packet, addr: data.Addr})
				}
			}

		case rpc := <-rpcch:
			p.sendPacket(rpc.Dst, "dht", rpc.Payload)

		case <-p.exit:
			return
		}
	}
}

func (p *node) sendMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, "msg", payload)
}

func (p *node) messageChannel() <-chan message {
	return p.msgChan
}

func (p *node) sendPublicKeyResponse(addr *net.UDPAddr) {
	data, _ := msgpack.Marshal(p.key.PublicKey)
	p.sendPacketAddr(addr, "key", data)
}

func (p *node) processPublicKeyResponse(packet packet) {
	var key PublicKey
	err := msgpack.Unmarshal(packet.Payload, &key)
	if err == nil {
		id := key.PublicKeyHash()
		if id.Cmp(packet.Src) == 0 {
			id := packet.Src.String()
			p.keycache[id] = key
			p.logger.Info("Get publickey for %s", id)
			p.processWaitingPackets()
		} else {
			p.logger.Error("receive wrong public key")
		}
	}
}

func (p *node) processPacket(packet packet, addr *net.UDPAddr) {
	info := nodeInfo{Id: packet.Src, Addr: addr}
	switch packet.Type {
	case "disco":
		p.dht.addNode(info)
	case "dht":
		p.dht.processPacket(info, packet.Payload)
	case "msg":
		p.msgChan <- message{id: info.Id, payload: packet.Payload}
	}
}

func (p *node) addWaitingPacket(in incomingPacket) {
	p.waitingPackets = append(p.waitingPackets, in)
}

// process packets waiting publickeys
func (p *node) processWaitingPackets() {
	rest := make([]incomingPacket, 0, len(p.keycache))
	for _, in := range p.waitingPackets {
		// find publickey from cache
		if key, ok := p.keycache[in.packet.Src.String()]; ok {
			if in.packet.verify(&key) {
				p.processPacket(in.packet, in.addr)
			}
		} else {
			rest = append(rest, in)
		}
	}
	p.waitingPackets = rest
}

func (p *node) sendDiscoveryPacket(addr *net.UDPAddr) {
	p.sendPacketAddr(addr, "disco", nil)
}

func (p *node) sendPacketAddr(addr *net.UDPAddr, typ string, payload []byte) {
	packet := packet{
		Src:     p.selfnode.Id,
		Type:    typ,
		Payload: payload,
	}

	packet.sign(p.key)

	data, err := msgpack.Marshal(packet)
	if err != nil {
		panic(err)
	}
	p.conn.WriteToUDP(data, addr)
}

func (p *node) sendPacket(dst NodeId, typ string, payload []byte) {
	packet := packet{
		Dst:     dst,
		Src:     p.selfnode.Id,
		Type:    typ,
		Payload: payload,
	}

	p.sendChan <- packet
	p.logger.Info("Send %s packet to %s", packet.Type, packet.Dst.String())
}

func (p *node) close() {
	p.dht.close()
	p.exit <- struct{}{}
	p.conn.Close()
}
