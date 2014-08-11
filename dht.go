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
	rpcChan  chan DhtPacket
	callback map[string]dhtRpcCallback
}

type dhtRpcCommand struct {
	Id     []byte                 `msgpack:"id"`
	Method string                 `msgpack:"method"`
	Args   map[string]interface{} `msgpack:"args"`
}

type DhtPacket struct {
	Dst     NodeId
	Payload []byte
}

func NewDht(k int, selfnode NodeInfo) *Dht {
	return &Dht{
		selfnode: selfnode,
		table:    NewNodeTable(k),
		rpcChan:  make(chan DhtPacket),
		callback: make(map[string]dhtRpcCallback),
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

func (p *Dht) RpcChannel() <-chan DhtPacket {
	return p.rpcChan
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
	go func() {
		p.rpcChan <- DhtPacket{Dst: dst, Payload: data}
	}()
}
