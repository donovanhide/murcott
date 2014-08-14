package murcott

import (
	"crypto/rand"
	"github.com/vmihailenco/msgpack"
	"net"
	"time"
)

type dhtRpcCallback func(*dhtRpcCommand, *net.UDPAddr)

type dhtPacket struct {
	Dst     NodeId
	Payload []byte
}

type dhtRpcReturn struct {
	command dhtRpcCommand
	addr    *net.UDPAddr
}

type dhtRpcRetunChan struct {
	id string
	ch chan *dhtRpcReturn
}

type dht struct {
	selfnode nodeInfo
	table    nodeTable
	rpcRet   map[string]chan *dhtRpcReturn
	rpcChan  chan dhtPacket
	logger   *Logger

	exit               chan struct{}
	addNodeRequest     chan nodeInfo
	updateNodeRequest  chan nodeInfo
	removeNodeRequest  chan NodeId
	nodeInfoRequest    chan NodeId
	nodeInfoResponse   chan *nodeInfo
	addRetChanRequest  chan dhtRpcRetunChan
	getRetChanRequest  chan string
	getRetChanResponse chan chan *dhtRpcReturn
}

type dhtRpcCommand struct {
	Id     []byte                 `msgpack:"id"`
	Method string                 `msgpack:"method"`
	Args   map[string]interface{} `msgpack:"args"`
}

func newDht(k int, selfnode nodeInfo, logger *Logger) *dht {
	d := dht{
		selfnode: selfnode,
		table:    newNodeTable(k),
		rpcRet:   make(map[string]chan *dhtRpcReturn),
		rpcChan:  make(chan dhtPacket, 100),
		logger:   logger,

		exit:               make(chan struct{}),
		addNodeRequest:     make(chan nodeInfo),
		updateNodeRequest:  make(chan nodeInfo),
		removeNodeRequest:  make(chan NodeId),
		nodeInfoRequest:    make(chan NodeId),
		nodeInfoResponse:   make(chan *nodeInfo),
		addRetChanRequest:  make(chan dhtRpcRetunChan, 100),
		getRetChanRequest:  make(chan string),
		getRetChanResponse: make(chan chan *dhtRpcReturn),
	}
	return &d
}

func (p *dht) run() {
	for {
		select {
		case node := <-p.addNodeRequest:
			p.table.insert(node, node.Id.Xor(p.selfnode.Id))
			p.sendPing(node.Id)
		case node := <-p.updateNodeRequest:
			p.table.insert(node, node.Id.Xor(p.selfnode.Id))
		case id := <-p.nodeInfoRequest:
			p.nodeInfoResponse <- p.table.find(id, id.Xor(p.selfnode.Id))
		case c := <-p.addRetChanRequest:
			p.rpcRet[c.id] = c.ch
		case id := <-p.getRetChanRequest:
			if ch, ok := p.rpcRet[id]; ok {
				delete(p.rpcRet, id)
				p.getRetChanResponse <- ch
			} else {
				p.getRetChanResponse <- nil
			}
		case <-p.exit:
			return
		}
	}
}

func (p *dht) close() {
	p.exit <- struct{}{}
}

func newRpcCommand(method string, args map[string]interface{}) dhtRpcCommand {
	id := make([]byte, 20)
	_, err := rand.Read(id)
	if err != nil {
		panic(err)
	}
	return dhtRpcCommand{
		Id:     id,
		Method: method,
		Args:   args,
	}
}

func newRpcReturnCommand(id []byte, args map[string]interface{}) dhtRpcCommand {
	return dhtRpcCommand{
		Id:     id,
		Method: "",
		Args:   args,
	}
}

func (p *dht) rpcChannel() <-chan dhtPacket {
	return p.rpcChan
}

func (p *dht) addNode(node nodeInfo) {
	p.addNodeRequest <- node
}

func (p *dht) updateNode(node nodeInfo) {
	p.updateNodeRequest <- node
}

func (p *dht) getNodeInfo(id NodeId) *nodeInfo {
	p.nodeInfoRequest <- id
	return <-p.nodeInfoResponse
}

func (p *dht) getRpcRetChan(id string) chan<- *dhtRpcReturn {
	p.getRetChanRequest <- id
	return <-p.getRetChanResponse
}

func (p *dht) sendPing(dst NodeId) {
	c := newRpcCommand("ping", nil)
	ch := p.sendPacket(dst, c)
	go func() {
		ret := <-ch
		if ret != nil {
			p.logger.Info("Receive DHT ping response")
		}
	}()
}

func (p *dht) sendPacket(dst NodeId, command dhtRpcCommand) chan *dhtRpcReturn {
	data, err := msgpack.Marshal(command)
	if err != nil {
		panic(err)
	}

	id := string(command.Id)
	ch := make(chan *dhtRpcReturn, 2)

	p.addRetChanRequest <- dhtRpcRetunChan{id: id, ch: ch}
	p.rpcChan <- dhtPacket{Dst: dst, Payload: data}

	go func() {
		// timeout
		<-time.After(200 * time.Millisecond)
		if ch := p.getRpcRetChan(id); ch != nil {
			ch <- nil
		}
	}()

	return ch
}

func (p *dht) processPacket(src nodeInfo, payload []byte, addr *net.UDPAddr) {
	var out dhtRpcCommand
	err := msgpack.Unmarshal(payload, &out)
	if err == nil {
		p.updateNode(src)
		switch out.Method {
		case "ping":
			p.logger.Info("Receive DHT Ping from %s", src.Id.String())
			p.sendPacket(src.Id, newRpcReturnCommand(out.Id, nil))

		case "": // callback
			id := string(out.Id)
			if ch := p.getRpcRetChan(id); ch != nil {
				ch <- &dhtRpcReturn{command: out, addr: addr}
			}
		}
	}
}
