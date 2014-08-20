package murcott

import (
	"errors"
	"github.com/vmihailenco/msgpack"
	"net"
)

const (
	portBegin = 9200
	portEnd   = 9220
	bootstrap = "127.0.0.1"
)

type nodeInfo struct {
	Id   NodeId
	Addr *net.UDPAddr
}

type message struct {
	id      NodeId
	payload []byte
}

type node struct {
	info        nodeInfo
	dht         *dht
	conn        *net.UDPConn
	key         *PrivateKey
	keycache    map[string]PublicKey
	keyWaiting  []packet
	addrWaiting []packet
	logger      *Logger
	recv        chan message
	send        chan packet
	exit        chan struct{}
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
	conn, selfport := getOpenPortConn()

	logger.Info("Node ID: %s", selfnode.Id.String())
	logger.Info("Node UDP port: %d", selfport)

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	node := node{
		info:     selfnode,
		conn:     conn,
		key:      key,
		keycache: make(map[string]PublicKey),
		dht:      dht,
		logger:   logger,
		recv:     make(chan message, 100),
		send:     make(chan packet, 100),
		exit:     exit,
	}

	// portscan
	for port := portBegin; port <= portEnd; port++ {
		if port != selfport {
			addr := &net.UDPAddr{Port: port, IP: host[0]}
			node.sendPacket(NodeId{}, addr, "disco", nil)
		}
	}

	logger.Info("Sent discovery packet to %v:%d-%d", host[0], portBegin, portEnd)
	go node.run()

	return &node
}

func (p *node) sendMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, nil, "msg", payload)
}

func (p *node) recvMessage() (NodeId, []byte, error) {
	if m, ok := <-p.recv; ok {
		return m.id, m.payload, nil
	} else {
		return NodeId{}, nil, errors.New("Node closed")
	}
}

func (p *node) run() {

	recv := make(chan packet)
	rpcch := p.dht.rpcChannel()

	// read datagram from udp socket
	go func() {
		for {
			var buf [1024]byte
			len, addr, err := p.conn.ReadFromUDP(buf[:])
			if err != nil {
				break
			}

			var packet packet
			err = msgpack.Unmarshal(buf[:len], &packet)
			if err != nil {
				continue
			}
			p.logger.Info("Receive %s packet from %s", packet.Type, packet.Src.String())
			packet.addr = addr

			recv <- packet
		}
	}()

	go p.dht.run()

	for {
		select {
		case packet := <-p.send:
			packet.sign(p.key)
			addr := packet.addr
			if addr == nil {
				node := p.dht.getNodeInfo(packet.Dst)
				if node != nil {
					addr = node.Addr
				}
			}
			if addr != nil {
				data, err := msgpack.Marshal(packet)
				if err == nil {
					p.conn.WriteToUDP(data, addr)
				} else {
					p.logger.Error("packet marshal error")
				}
			} else {
				p.logger.Error("route not found: %s", packet.Dst.String())
				p.addrWaiting = append(p.addrWaiting, packet)
			}

		case packet := <-recv:
			if packet.Type == "key" {
				if len(packet.Payload) == 0 {
					key, _ := msgpack.Marshal(p.key.PublicKey)
					p.sendPacket(packet.Src, packet.addr, "key", key)
				} else {
					p.processPublicKeyResponse(packet)
				}
			} else {
				// find publickey from cache
				if key, ok := p.keycache[packet.Src.String()]; ok {
					if packet.verify(&key) {
						p.processPacket(packet)
					}
				} else {
					// request publickey
					p.sendPacket(packet.Src, packet.addr, "key", nil)
					p.keyWaiting = append(p.keyWaiting, packet)
				}
			}
			p.processWaitingRouteyPackets()

		case rpc := <-rpcch:
			p.sendPacket(rpc.Dst, nil, "dht", rpc.Payload)

		case <-p.exit:
			return
		}
	}
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
			p.processWaitingKeyPackets()
		} else {
			p.logger.Error("receive wrong public key")
		}
	}
}

func (p *node) processPacket(packet packet) {
	info := nodeInfo{Id: packet.Src, Addr: packet.addr}
	switch packet.Type {
	case "disco":
		p.dht.addNode(info)
	case "dht":
		p.dht.processPacket(info, packet.Payload)
	case "msg":
		p.recv <- message{id: info.Id, payload: packet.Payload}
	}
}

// process packets waiting publickeys
func (p *node) processWaitingKeyPackets() {
	rest := make([]packet, 0, len(p.keyWaiting))
	for _, packet := range p.keyWaiting {
		// find publickey from cache
		if key, ok := p.keycache[packet.Src.String()]; ok {
			if packet.verify(&key) {
				p.processPacket(packet)
			}
		} else {
			rest = append(rest, packet)
		}
	}
	p.keyWaiting = rest
}

// process packets waiting addresses
func (p *node) processWaitingRouteyPackets() {
	rest := make([]packet, 0, len(p.addrWaiting))
	for _, packet := range p.addrWaiting {
		node := p.dht.getNodeInfo(packet.Dst)
		if node != nil {
			data, err := msgpack.Marshal(packet)
			if err == nil {
				p.conn.WriteToUDP(data, node.Addr)
			} else {
				p.logger.Error("packet marshal error")
			}
		} else {
			rest = append(rest, packet)
		}
	}
	p.addrWaiting = rest
}

func (p *node) sendPacket(dst NodeId, addr *net.UDPAddr, typ string, payload []byte) {
	packet := packet{
		Dst:     dst,
		Src:     p.info.Id,
		Type:    typ,
		Payload: payload,
		addr:    addr,
	}
	p.send <- packet

	if id := dst.String(); len(id) > 0 {
		p.logger.Info("Send %s packet to %s", packet.Type, id)
	}
}

func (p *node) close() {
	close(p.recv)
	p.dht.close()
	p.exit <- struct{}{}
	p.conn.Close()
}
