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

type node struct {
	selfnode nodeInfo
	dht      *dht
	conn     *net.UDPConn
	exitch   chan struct{}
	logger   *Logger
	msgcb    []func(NodeId, []byte)
}

type nodeInfo struct {
	Id   NodeId
	Addr *net.UDPAddr
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

func newNode(logger *Logger) *node {
	selfnode := nodeInfo{Id: *NewRandomNodeId()}
	conn, selfport := getOpenPortConn()
	dht := newDht(10, selfnode, logger)
	exitch := make(chan struct{})

	logger.Info("Node ID: %s", selfnode.Id.String())
	logger.Info("Node UDP port: %d", selfport)

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	node := node{
		selfnode: selfnode,
		conn:     conn,
		dht:      dht,
		exitch:   exitch,
		logger:   logger,
	}

	// portscan
	for port := portBegin; port <= portEnd; port++ {
		if port != selfport {
			node.sendDiscoveryPacket(net.UDPAddr{Port: port, IP: host[0]})
		}
	}

	logger.Info("Sent discovery packet to %v:%d-%d", host[0], portBegin, portEnd)

	datach := make(chan udpDatagram)

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

	dht.rpcCallback(func(dst NodeId, payload []byte) {
		node.sendPacket(dst, "dht", payload)
	})

	go func() {
		for {
			select {
			case data := <-datach:
				var out packet
				err = msgpack.Unmarshal(data.Data, &out)
				if err == nil {
					node.logger.Info("Receive %s packet from %s", out.Type, out.Src.String())
					node.processPacket(out, data.Addr)
				}
			case <-exitch:
				break
			}
		}
	}()

	return &node
}

func (p *node) sendMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, "msg", payload)
}

func (p *node) messageCallback(cb func(NodeId, []byte)) {
	p.msgcb = append(p.msgcb, cb)
}

func (p *node) processPacket(packet packet, addr *net.UDPAddr) {
	info := nodeInfo{Id: packet.Src, Addr: addr}
	switch packet.Type {
	case "disco":
		p.dht.addNode(info)
	case "dht":
		p.dht.processPacket(info, packet.Payload, addr)
	case "msg":
		for _, cb := range p.msgcb {
			cb(info.Id, packet.Payload)
		}
	}
}

func (p *node) sendDiscoveryPacket(addr net.UDPAddr) {
	packet := packet{
		Src:  p.selfnode.Id,
		Type: "disco",
	}
	data, err := msgpack.Marshal(packet)
	if err != nil {
		panic(err)
	}
	p.conn.WriteToUDP(data, &addr)
}

func (p *node) sendPacket(dst NodeId, typ string, payload []byte) {
	packet := packet{
		Dst:     dst,
		Src:     p.selfnode.Id,
		Type:    typ,
		Payload: payload,
	}

	data, err := msgpack.Marshal(packet)
	if err != nil {
		panic(err)
	}

	node := p.dht.getNodeInfo(dst)

	p.logger.Info("Send %s packet to %s", packet.Type, packet.Dst.String())

	if node != nil {
		p.conn.WriteToUDP(data, node.Addr)
	}
}

func (p *node) id() NodeId {
	return p.selfnode.Id
}

func (p *node) close() {
	p.exitch <- struct{}{}
	p.conn.Close()
}
