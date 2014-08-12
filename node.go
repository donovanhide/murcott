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

type Node struct {
	selfnode NodeInfo
	dht      *Dht
	conn     *net.UDPConn
	exitch   chan struct{}
	logger   *Logger
	msgcb    []func(NodeId, []byte)
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

func NewNode(logger *Logger) *Node {
	selfnode := NodeInfo{Id: *NewRandomNodeId()}
	conn, selfport := getOpenPortConn()
	dht := NewDht(10, selfnode, logger)
	exitch := make(chan struct{})

	logger.Info("Node ID: %s", selfnode.Id.String())
	logger.Info("Node UDP port: %d", selfport)

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	node := Node{
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

	dht.RpcCallback(func(dst NodeId, payload []byte) {
		node.sendPacket(dst, "dht", payload)
	})

	go func() {
		for {
			select {
			case data := <-datach:
				var out Packet
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

func (p *Node) SendMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, "msg", payload)
}

func (p *Node) MessageCallback(cb func(NodeId, []byte)) {
	p.msgcb = append(p.msgcb, cb)
}

func (p *Node) processPacket(packet Packet, addr *net.UDPAddr) {
	info := NodeInfo{Id: packet.Src, Addr: addr}
	switch packet.Type {
	case "disco":
		p.dht.AddNode(info)
	case "dht":
		p.dht.ProcessPacket(info, packet.Payload, addr)
	case "msg":
		for _, cb := range p.msgcb {
			cb(info.Id, packet.Payload)
		}
	}
}

func (p *Node) sendDiscoveryPacket(addr net.UDPAddr) {
	packet := Packet{
		Src:  p.selfnode.Id,
		Type: "disco",
	}
	data, err := msgpack.Marshal(packet)
	if err != nil {
		panic(err)
	}
	p.conn.WriteToUDP(data, &addr)
}

func (p *Node) sendPacket(dst NodeId, typ string, payload []byte) {
	packet := Packet{
		Dst:     dst,
		Src:     p.selfnode.Id,
		Type:    typ,
		Payload: payload,
	}

	data, err := msgpack.Marshal(packet)
	if err != nil {
		panic(err)
	}

	node := p.dht.GetNodeInfo(dst)

	p.logger.Info("Send %s packet to %s", packet.Type, packet.Dst.String())

	if node != nil {
		p.conn.WriteToUDP(data, node.Addr)
	}
}

func (p *Node) Id() NodeId {
	return p.selfnode.Id
}

func (p *Node) Close() {
	p.exitch <- struct{}{}
	p.conn.Close()
}
