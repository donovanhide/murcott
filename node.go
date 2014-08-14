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

type message struct {
	id      NodeId
	payload []byte
}

type node struct {
	selfnode nodeInfo
	dht      *dht
	conn     *net.UDPConn
	logger   *Logger
	msgcb    []func(NodeId, []byte)
	msgChan  chan message
	exit     chan struct{}
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
	dht := newDht(10, selfnode, logger)
	exit := make(chan struct{})

	node := node{
		selfnode: selfnode,
		conn:     nil,
		dht:      dht,
		logger:   logger,
		msgChan:  make(chan message),
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
			p.sendDiscoveryPacket(net.UDPAddr{Port: port, IP: host[0]})
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
		case data := <-datach:
			var out packet
			err = msgpack.Unmarshal(data.Data, &out)
			if err == nil {
				p.logger.Info("Receive %s packet from %s", out.Type, out.Src.String())
				p.processPacket(out, data.Addr)
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

func (p *node) processPacket(packet packet, addr *net.UDPAddr) {
	info := nodeInfo{Id: packet.Src, Addr: addr}
	switch packet.Type {
	case "disco":
		p.dht.addNode(info)
	case "dht":
		p.dht.processPacket(info, packet.Payload, addr)
	case "msg":
		p.msgChan <- message{id: info.Id, payload: packet.Payload}
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
	p.dht.close()
	p.exit <- struct{}{}
	p.conn.Close()
}
