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

func NewNode() *Node {
	selfnode := NodeInfo{Id: *NewRandomNodeId()}
	conn, selfport := getOpenPortConn()
	dht := NewDht(10, selfnode)
	exitch := make(chan struct{})

	// lookup bootstrap
	host, err := net.LookupIP(bootstrap)
	if err != nil {
		panic(err)
	}

	Node := Node{
		selfnode: selfnode,
		conn:     conn,
		dht:      dht,
		exitch:   exitch,
	}

	// portscan
	for port := portBegin; port <= portEnd; port++ {
		if port != selfport {
			Node.sendDiscoveryPacket(net.UDPAddr{Port: port, IP: host[0]})
		}
	}

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

	go func() {
		dhtch := dht.RpcChannel()
		for {
			select {
			case data := <-datach:
				var out Packet
				err = msgpack.Unmarshal(data.Data, &out)
				if err == nil {
					Node.processPacket(out, data.Addr)
				}
			case packet := <-dhtch:
				Node.sendPacket(packet.Dst, "dht", packet.Payload)
			case <-exitch:
				break
			}
		}
	}()

	return &Node
}

func (p *Node) SendRawMessage(dst NodeId, payload []byte) {
	p.sendPacket(dst, "msg", payload)
}

func (p *Node) processPacket(packet Packet, addr *net.UDPAddr) {
	info := NodeInfo{Id: packet.Src, Addr: addr}
	switch packet.Type {
	case "disco":
		p.dht.AddNode(info)
	case "dht":
		p.dht.ProcessPacket(info, packet.Payload, addr)
	case "msg":
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
