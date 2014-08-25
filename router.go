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

type message struct {
	id      NodeId
	payload []byte
}

type router struct {
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

func newRouter(key *PrivateKey, logger *Logger) *router {
	info := nodeInfo{Id: key.PublicKeyHash()}
	dht := newDht(10, info, logger)
	exit := make(chan struct{})
	conn, selfport := getOpenPortConn()

	logger.info("Node ID: %s", info.Id.String())
	logger.info("Node UDP port: %d", selfport)

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	node := router{
		info:     info,
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

	logger.info("Sent discovery packet to %v:%d-%d", host[0], portBegin, portEnd)
	go node.run()

	return &node
}

func (p *router) sendMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, nil, "msg", payload)
}

func (p *router) recvMessage() (message, error) {
	if m, ok := <-p.recv; ok {
		return m, nil
	} else {
		return message{}, errors.New("Node closed")
	}
}

func (p *router) run() {

	recv := make(chan packet)

	// read datagram from udp socket
	go func() {
		for {
			var buf [65507]byte
			len, addr, err := p.conn.ReadFromUDP(buf[:])
			if err != nil {
				break
			}

			var packet packet
			err = msgpack.Unmarshal(buf[:len], &packet)
			if err != nil {
				continue
			}
			p.logger.info("Receive %s packet from %s", packet.Type, packet.Src.String())
			packet.addr = addr

			recv <- packet
		}
	}()

	go func() {
		for {
			dst, payload, err := p.dht.nextPacket()
			if err != nil {
				return
			}
			p.sendPacket(dst, nil, "dht", payload)
		}
	}()

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
					p.logger.error("packet marshal error")
				}
			} else {
				p.logger.error("route not found: %s", packet.Dst.String())
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
			p.processWaitingRoutePackets()

		case <-p.exit:
			return
		}
	}
}

func (p *router) processPublicKeyResponse(packet packet) {
	var key PublicKey
	err := msgpack.Unmarshal(packet.Payload, &key)
	if err == nil {
		id := key.PublicKeyHash()
		if id.cmp(packet.Src) == 0 {
			id := packet.Src.String()
			p.keycache[id] = key
			p.logger.info("Get publickey for %s", id)
			p.processWaitingKeyPackets()
		} else {
			p.logger.error("receive wrong public key")
		}
	}
}

func (p *router) processPacket(packet packet) {
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
func (p *router) processWaitingKeyPackets() {
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
func (p *router) processWaitingRoutePackets() {
	rest := make([]packet, 0, len(p.addrWaiting))
	for _, packet := range p.addrWaiting {
		node := p.dht.getNodeInfo(packet.Dst)
		if node != nil {
			data, err := msgpack.Marshal(packet)
			if err == nil {
				p.conn.WriteToUDP(data, node.Addr)
			} else {
				p.logger.error("packet marshal error")
			}
		} else {
			rest = append(rest, packet)
		}
	}
	p.addrWaiting = rest
}

func (p *router) sendPacket(dst NodeId, addr *net.UDPAddr, typ string, payload []byte) {
	packet := packet{
		Dst:     dst,
		Src:     p.info.Id,
		Type:    typ,
		Payload: payload,
		addr:    addr,
	}
	p.send <- packet

	if id := dst.String(); len(id) > 0 {
		p.logger.info("Send %s packet to %s", packet.Type, id)
	}
}

func (p *router) close() {
	p.exit <- struct{}{}
	close(p.recv)
	p.dht.close()
	p.conn.Close()
}
