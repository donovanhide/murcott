package murcott

import (
	"errors"
	"net"
	"strconv"

	"github.com/vmihailenco/msgpack"
)

type message struct {
	id      NodeId
	payload []byte
}

type queuedPacket struct {
	id     int
	packet *packet
}

type router struct {
	info        nodeInfo
	dht         *dht
	conn        *net.UDPConn
	key         *PrivateKey
	keycache    map[string]PublicKey
	keyWaiting  []packet
	addrWaiting map[int]packet
	logger      *Logger
	packetId    chan int
	recv        chan message
	send        chan queuedPacket
	exit        chan int
}

func getOpenPortConn(config Config) (*net.UDPConn, int) {
	for _, port := range config.getPorts() {
		addr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(port))
		conn, err := net.ListenUDP("udp", addr)
		if err == nil {
			return conn, port
		}
	}
	return nil, 0
}

func newRouter(key *PrivateKey, logger *Logger, config Config) *router {
	info := nodeInfo{Id: key.PublicKeyHash()}
	dht := newDht(10, info, logger)
	exit := make(chan int)
	conn, selfport := getOpenPortConn(config)

	logger.info("Node ID: %s", info.Id.String())
	logger.info("Node UDP port: %d", selfport)

	r := router{
		info:        info,
		conn:        conn,
		key:         key,
		keycache:    make(map[string]PublicKey),
		dht:         dht,
		addrWaiting: make(map[int]packet),
		logger:      logger,
		packetId:    make(chan int),
		recv:        make(chan message, 100),
		send:        make(chan queuedPacket, 100),
		exit:        exit,
	}

	go func() {
		packetId := 0
		for {
			r.packetId <- packetId
			packetId++
		}
	}()

	// portscan
	for _, addr := range config.getBootstrap() {
		a := addr
		r.sendPacket(NodeId{}, &a, "disco", nil)
		logger.info("Sent discovery packet to %v:%d", addr.IP, addr.Port)
	}

	go r.run()

	return &r
}

func (p *router) sendMessage(dst NodeId, payload []byte) int {
	return p.sendPacket(dst, nil, "msg", payload)
}

func (p *router) cancelMessage(id int) {
	p.send <- queuedPacket{id: id, packet: nil}
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

			if packet.Src.cmp(p.info.Id) == 0 {
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
		case q := <-p.send:
			if q.packet == nil {
				// cancel queued packet
				delete(p.addrWaiting, q.id)
			} else {
				q.packet.sign(p.key)
				addr := q.packet.addr
				if addr != nil {
					data, err := msgpack.Marshal(q.packet)
					if err == nil {
						p.conn.WriteToUDP(data, addr)
					} else {
						p.logger.error("packet marshal error")
					}
				} else {
					p.addrWaiting[q.id] = *q.packet
				}
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

		case <-p.exit:
			return
		}
		p.processWaitingRoutePackets()
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
	for id, packet := range p.addrWaiting {
		node := p.dht.getNodeInfo(packet.Dst)
		if node != nil {
			data, err := msgpack.Marshal(packet)
			if err == nil {
				p.conn.WriteToUDP(data, node.Addr)
			} else {
				p.logger.error("packet marshal error")
			}
			delete(p.addrWaiting, id)
		}
	}
}

func (p *router) sendPacket(dst NodeId, addr *net.UDPAddr, typ string, payload []byte) int {
	packet := packet{
		Dst:     dst,
		Src:     p.info.Id,
		Type:    typ,
		Payload: payload,
		addr:    addr,
	}

	id := <-p.packetId
	p.send <- queuedPacket{id: id, packet: &packet}

	if d := dst.String(); len(d) > 0 {
		p.logger.info("Send %s packet to %s", packet.Type, d)
	}

	return id
}

func (p *router) addNode(info nodeInfo) {
	p.dht.addNode(info)
}

func (p *router) knownNodes() []nodeInfo {
	return p.dht.knownNodes()
}

func (p *router) close() {
	p.exit <- 0
	close(p.recv)
	p.dht.close()
	p.conn.Close()
}
