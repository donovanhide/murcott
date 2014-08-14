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
		table:    newNodeTable(k, selfnode.Id),
		rpcRet:   make(map[string]chan *dhtRpcReturn),
		rpcChan:  make(chan dhtPacket, 100),
		logger:   logger,

		exit:               make(chan struct{}),
		addRetChanRequest:  make(chan dhtRpcRetunChan, 100),
		getRetChanRequest:  make(chan string),
		getRetChanResponse: make(chan chan *dhtRpcReturn),
	}
	return &d
}

func (p *dht) run() {
	for {
		select {
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
	p.table.insert(node)
	p.sendPing(node.Id)
}

func (p *dht) updateNode(node nodeInfo) {
	p.table.insert(node)
}

func (p *dht) getNodeInfo(id NodeId) *nodeInfo {
	return p.table.find(id)
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

	if len(command.Method) > 0 {
		go func() {
			// timeout
			<-time.After(200 * time.Millisecond)
			if ch := p.getRpcRetChan(id); ch != nil {
				ch <- nil
				p.table.remove(dst)
				p.logger.Info("Remove %s from routing table", dst.String())
			}
		}()
	}

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
