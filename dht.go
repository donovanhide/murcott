package murcott

import (
	"crypto/rand"
	"github.com/vmihailenco/msgpack"
	"net"
)

type dhtRpcCallback func(*dhtRpcCommand, *net.UDPAddr)

type Dht struct {
	selfnode NodeInfo
	table    NodeTable
	callback map[string]dhtRpcCallback
	rpccb    []func(NodeId, []byte)
	logger   *Logger
}

type dhtRpcCommand struct {
	Id     []byte                 `msgpack:"id"`
	Method string                 `msgpack:"method"`
	Args   map[string]interface{} `msgpack:"args"`
}

func NewDht(k int, selfnode NodeInfo, logger *Logger) *Dht {
	return &Dht{
		selfnode: selfnode,
		table:    NewNodeTable(k),
		callback: make(map[string]dhtRpcCallback),
		rpccb:    make([]func(NodeId, []byte), 0),
		logger:   logger,
	}
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

func (p *Dht) RpcCallback(cb func(dst NodeId, payload []byte)) {
	p.rpccb = append(p.rpccb, cb)
}

func (p *Dht) AddNode(node NodeInfo) {
	p.table.Insert(node, node.Id.Xor(p.selfnode.Id))
	p.sendPing(node.Id)
}

func (p *Dht) sendPing(dst NodeId) {
	c := newRpcCommand("ping", nil)
	p.sendPacket(dst, c, func(packet *dhtRpcCommand, addr *net.UDPAddr) {
		if packet == nil {
			// TODO: remove entry
		}
	})
}

func (p *Dht) GetNodeInfo(id NodeId) *NodeInfo {
	return p.table.Find(id, id.Xor(p.selfnode.Id))
}

func (p *Dht) ProcessPacket(src NodeInfo, payload []byte, addr *net.UDPAddr) {
	var out dhtRpcCommand
	err := msgpack.Unmarshal(payload, &out)

	if err == nil {
		p.table.Insert(src, src.Id.Xor(p.selfnode.Id))
		switch out.Method {
		case "ping":
			p.logger.Info("Receive DHT Ping from %s", src.Id.String())
			p.sendPacket(src.Id, newRpcReturnCommand(out.Id, nil), nil)

		case "": // callback
			if f, ok := p.callback[string(out.Id)]; ok {
				f(&out, addr)
				delete(p.callback, string(out.Id))
			}
		}
	}
}

func (p *Dht) sendPacket(dst NodeId, command dhtRpcCommand, cb dhtRpcCallback) {
	data, err := msgpack.Marshal(command)
	if err != nil {
		panic(err)
	}
	p.callback[string(command.Id)] = cb
	for _, cb := range p.rpccb {
		cb(dst, data)
	}
}
