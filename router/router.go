package router

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/h2so5/murcott/dht"
	"github.com/h2so5/murcott/internal"
	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/utils"
	"github.com/h2so5/utp"
	"github.com/vmihailenco/msgpack"
)

type Message struct {
	ID      utils.NodeID
	Payload []byte
}

type queuedPacket struct {
	id     int
	packet *internal.Packet
}

type Router struct {
	info           utils.NodeInfo
	dht            *dht.DHT
	conn           *net.UDPConn
	c              *utp.Conn
	key            *utils.PrivateKey
	keycache       map[string]utils.PublicKey
	keyWaiting     []internal.Packet
	addrWaiting    map[int]internal.Packet
	requestedNodes map[string]time.Time
	logger         *log.Logger
	packetID       chan int
	recv           chan Message
	send           chan queuedPacket
	exit           chan int
}

func getOpenPortConn(config utils.Config) (*net.UDPConn, int, error) {
	for _, port := range config.Ports() {
		addr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(port))
		conn, err := net.ListenUDP("udp", addr)
		if err == nil {
			return conn, port, nil
		}
	}
	return nil, 0, errors.New("fail to bind port")
}

func NewRouter(key *utils.PrivateKey, logger *log.Logger, config utils.Config) (*Router, error) {
	info := utils.NodeInfo{ID: key.PublicKeyHash()}
	dht := dht.NewDHT(10, info, logger)
	exit := make(chan int)
	conn, selfport, err := getOpenPortConn(config)
	if err != nil {
		return nil, err
	}

	logger.Info("Node ID: %s", info.ID.String())
	logger.Info("Node UDP port: %d", selfport)

	r := Router{
		info:           info,
		conn:           conn,
		key:            key,
		keycache:       make(map[string]utils.PublicKey),
		dht:            dht,
		addrWaiting:    make(map[int]internal.Packet),
		requestedNodes: make(map[string]time.Time),
		logger:         logger,
		packetID:       make(chan int),
		recv:           make(chan Message, 100),
		send:           make(chan queuedPacket, 100),
		exit:           exit,
	}

	go func() {
		packetID := 0
		for {
			r.packetID <- packetID
			packetID++
		}
	}()

	go r.run()

	return &r, nil
}

func (p *Router) Discover(addrs []net.UDPAddr) {
	for _, addr := range addrs {
		a := addr
		p.sendPacket(utils.NodeID{}, &a, "disco", nil)
		p.logger.Info("Sent discovery packet to %v:%d", addr.IP, addr.Port)
	}
}

func (p *Router) SendMessage(dst utils.NodeID, payload []byte) int {
	return p.sendPacket(dst, nil, "msg", payload)
}

func (p *Router) CancelMessage(id int) {
	p.send <- queuedPacket{id: id, packet: nil}
}

func (p *Router) RecvMessage() (Message, error) {
	if m, ok := <-p.recv; ok {
		return m, nil
	}
	return Message{}, errors.New("Node closed")
}

func (p *Router) run() {

	recv := make(chan internal.Packet)

	// read datagram from udp socket
	go func() {
		for {
			var buf [65507]byte
			len, addr, err := p.conn.ReadFromUDP(buf[:])
			if err != nil {
				break
			}

			var packet internal.Packet
			err = msgpack.Unmarshal(buf[:len], &packet)
			if err != nil {
				continue
			}

			if packet.Src.Cmp(p.info.ID) == 0 {
				continue
			}

			p.logger.Info("Receive %s packet from %s", packet.Type, packet.Src.String())
			packet.Addr = addr

			recv <- packet
		}
	}()

	go func() {
		for {
			dst, payload, err := p.dht.NextPacket()
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
				q.packet.Sign(p.key)
				addr := q.packet.Addr
				if addr != nil {
					data, err := msgpack.Marshal(q.packet)
					if err == nil {
						p.conn.WriteToUDP(data, addr)
					} else {
						p.logger.Error("packet marshal error")
					}
				} else {
					p.addrWaiting[q.id] = *q.packet
				}
			}

		case packet := <-recv:
			if packet.Type == "key" {
				if len(packet.Payload) == 0 {
					key, _ := msgpack.Marshal(p.key.PublicKey)
					p.sendPacket(packet.Src, packet.Addr, "key", key)
				} else {
					p.processPublicKeyResponse(packet)
				}
			} else {
				// find publickey from cache
				if key, ok := p.keycache[packet.Src.String()]; ok {
					if packet.Verify(&key) {
						p.processPacket(packet)
					}
				} else {
					// request publickey
					p.sendPacket(packet.Src, packet.Addr, "key", nil)
					p.keyWaiting = append(p.keyWaiting, packet)
				}
			}
		case <-time.After(time.Second):
		case <-p.exit:
			return
		}
		p.processWaitingRoutePackets()
	}
}

func (p *Router) processPublicKeyResponse(packet internal.Packet) {
	var key utils.PublicKey
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

func (p *Router) processPacket(packet internal.Packet) {
	info := utils.NodeInfo{ID: packet.Src, Addr: packet.Addr}
	switch packet.Type {
	case "disco":
		p.dht.AddNode(info)
	case "dht":
		p.dht.ProcessPacket(packet)
	case "msg":
		p.recv <- Message{ID: info.ID, Payload: packet.Payload}
	}
}

// process packets waiting publickeys
func (p *Router) processWaitingKeyPackets() {
	rest := make([]internal.Packet, 0, len(p.keyWaiting))
	for _, packet := range p.keyWaiting {
		// find publickey from cache
		if key, ok := p.keycache[packet.Src.String()]; ok {
			if packet.Verify(&key) {
				p.processPacket(packet)
			}
		} else {
			rest = append(rest, packet)
		}
	}
	p.keyWaiting = rest
}

// process packets waiting addresses
func (p *Router) processWaitingRoutePackets() {
	var unknownNodes []utils.NodeID
	for id, packet := range p.addrWaiting {
		node := p.dht.GetNodeInfo(packet.Dst)
		if node != nil {
			data, err := msgpack.Marshal(packet)
			if err == nil {
				p.conn.WriteToUDP(data, node.Addr)
			} else {
				p.logger.Error("packet marshal error")
			}
			delete(p.addrWaiting, id)
		} else {
			unknownNodes = append(unknownNodes, packet.Dst)
		}
	}

	// Remove old entries.
	for k, v := range p.requestedNodes {
		if time.Since(v).Minutes() >= 5 {
			delete(p.requestedNodes, k)
		}
	}

	for _, n := range unknownNodes {
		if _, ok := p.requestedNodes[n.String()]; !ok {
			go p.dht.FindNearestNode(n)
			p.requestedNodes[n.String()] = time.Now()
		}
	}
}

func (p *Router) sendPacket(dst utils.NodeID, addr *net.UDPAddr, typ string, payload []byte) int {
	packet := internal.Packet{
		Dst:     dst,
		Src:     p.info.ID,
		Type:    typ,
		Payload: payload,
		Addr:    addr,
	}

	id := <-p.packetID
	p.send <- queuedPacket{id: id, packet: &packet}

	if d := dst.String(); len(d) > 0 {
		p.logger.Info("Send %s packet to %s", packet.Type, d)
	}

	return id
}

func (p *Router) AddNode(info utils.NodeInfo) {
	p.dht.AddNode(info)
}

func (p *Router) KnownNodes() []utils.NodeInfo {
	return p.dht.KnownNodes()
}

func (p *Router) Close() {
	p.exit <- 0
	close(p.recv)
	p.dht.Close()
	p.conn.Close()
}
