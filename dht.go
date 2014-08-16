package murcott

import (
	"crypto/rand"
	"crypto/sha1"
	"github.com/vmihailenco/msgpack"
	"net"
	"sort"
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
	k        int
	kvs      map[string]string
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

func (p *dhtRpcCommand) getArgs(k string, v ...interface{}) {
	b, err := msgpack.Marshal(p.Args[k])
	if err == nil {
		msgpack.Unmarshal(b, v...)
	}
}

func newDht(k int, selfnode nodeInfo, logger *Logger) *dht {
	d := dht{
		selfnode: selfnode,
		table:    newNodeTable(k, selfnode.Id),
		k:        k,
		kvs:      make(map[string]string),
		rpcRet:   make(map[string]chan *dhtRpcReturn),
		rpcChan:  make(chan dhtPacket, 100),
		logger:   logger,

		exit:               make(chan struct{}),
		addRetChanRequest:  make(chan dhtRpcRetunChan, 100),
		getRetChanRequest:  make(chan string, 100),
		getRetChanResponse: make(chan chan *dhtRpcReturn, 100),
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

func (p *dht) rpcChannel() <-chan dhtPacket {
	return p.rpcChan
}

func (p *dht) addNode(node nodeInfo) {
	p.table.insert(node)
	p.sendPing(node.Id)
}

func (p *dht) getNodeInfo(id NodeId) *nodeInfo {
	return p.table.find(id)
}

func (p *dht) storeValue(key string, value string) {
	hash := sha1.Sum([]byte(key))
	c := newRpcCommand("store", map[string]interface{}{
		"key":   key,
		"value": value,
	})
	for _, n := range p.findNearestNode(NewNodeId(hash[:])) {
		p.sendPacket(n.Id, c)
	}
}

type nodeInfoSorter struct {
	nodes []nodeInfo
	id    NodeId
}

func (p nodeInfoSorter) Len() int {
	return len(p.nodes)
}

func (p nodeInfoSorter) Swap(i, j int) {
	p.nodes[i], p.nodes[j] = p.nodes[j], p.nodes[i]
}

func (p nodeInfoSorter) Less(i, j int) bool {
	disti := p.nodes[i].Id.Xor(p.id)
	distj := p.nodes[j].Id.Xor(p.id)
	return (disti.Cmp(distj) == -1)
}

func (p *dht) findNearestNode(findid NodeId) []nodeInfo {

	reqch := make(chan nodeInfo, 100)
	endch := make(chan struct{}, 100)

	f := func(id NodeId, command dhtRpcCommand) {
		ch := p.sendPacket(id, command)

		ret := <-ch
		if ret != nil {
			if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []map[string]string
				ret.command.getArgs("nodes", &nodes)
				for _, n := range nodes {
					if id, ok := n["id"]; ok {
						if addr, err := net.ResolveUDPAddr("udp", n["addr"]); err == nil {
							node := nodeInfo{Id: NewNodeId([]byte(id)), Addr: addr}
							if node.Id.Cmp(p.selfnode.Id) != 0 {
								p.table.insert(node)
								reqch <- node
							}
						}
					}
				}
			}
		}
		endch <- struct{}{}
	}

	res := make([]nodeInfo, 0)
	nodes := p.table.nearestNodes(findid)

	if len(nodes) == 0 {
		return res
	}

	for _, n := range nodes {
		reqch <- n
	}

	count := 0
	requested := make(map[string]nodeInfo)

loop:
	for {
		select {
		case node := <-reqch:
			if _, ok := requested[node.Id.String()]; !ok {
				requested[node.Id.String()] = node
				c := newRpcCommand("find-node", map[string]interface{}{
					"id": string(findid.Bytes()),
				})
				go f(node.Id, c)
				count++
			}
		case <-endch:
			count--
			if count == 0 {
				break loop
			}
		}
	}

	for _, v := range requested {
		res = append(res, v)
	}

	sorter := nodeInfoSorter{nodes: res, id: findid}
	sort.Sort(sorter)

	if len(sorter.nodes) > p.k {
		return sorter.nodes[:p.k]
	} else {
		return sorter.nodes
	}
}

func (p *dht) loadValue(key string) *string {

	if v, ok := p.kvs[key]; ok {
		return &v
	}

	hash := sha1.Sum([]byte(key))
	keyid := NewNodeId(hash[:])

	retch := make(chan *string, 2)
	reqch := make(chan NodeId, 100)
	endch := make(chan struct{}, 100)

	nodes := p.table.nearestNodes(NewNodeId(hash[:]))

	f := func(id NodeId, keyid NodeId, command dhtRpcCommand) {
		ch := p.sendPacket(id, command)

		ret := <-ch
		if ret != nil {
			if val, ok := ret.command.Args["value"].(string); ok {
				retch <- &val
			} else if _, ok := ret.command.Args["nodes"]; ok {

				var nodes []map[string]string
				ret.command.getArgs("nodes", &nodes)
				dist := id.Xor(keyid)
				for _, n := range nodes {
					if id, ok := n["id"]; ok {
						if addr, err := net.ResolveUDPAddr("udp", n["addr"]); err == nil {
							node := NewNodeId([]byte(id))
							p.table.insert(nodeInfo{Id: node, Addr: addr})
							if dist.Cmp(node.Xor(keyid)) == 1 {
								reqch <- node
							}
						}
					}
				}
			}
		}
		endch <- struct{}{}
	}

	if len(nodes) == 0 {
		return nil
	}

	for _, n := range nodes {
		reqch <- n.Id
	}

	count := 0
	requested := make(map[string]struct{})

	for {
		select {
		case id := <-reqch:
			if _, ok := requested[id.String()]; !ok {
				requested[id.String()] = struct{}{}
				c := newRpcCommand("find-value", map[string]interface{}{
					"key": key,
				})
				go f(id, keyid, c)
				count++
			}
		case <-endch:
			count--
			if count == 0 {
				select {
				case data := <-retch:
					return data
				default:
					return nil
				}
			}
		case data := <-retch:
			return data
		default:
		}
	}
}

func (p *dht) processPacket(src nodeInfo, payload []byte, addr *net.UDPAddr) {
	var command dhtRpcCommand
	err := msgpack.Unmarshal(payload, &command)
	if err == nil {
		p.table.insert(src)

		switch command.Method {
		case "ping":
			p.logger.Info("Receive DHT Ping from %s", src.Id.String())
			p.sendPacket(src.Id, newRpcReturnCommand(command.Id, nil))

		case "find-node":
			p.logger.Info("Receive DHT Find-Node from %s", src.Id.String())
			if id, ok := command.Args["id"].(string); ok {
				args := map[string]interface{}{}
				nodes := p.table.nearestNodes(NewNodeId([]byte(id)))
				ary := make([]map[string]string, len(nodes))
				for i, n := range nodes {
					ary[i] = map[string]string{
						"id":   string(n.Id.Bytes()),
						"addr": n.Addr.String(),
					}
				}
				args["nodes"] = ary
				p.sendPacket(src.Id, newRpcReturnCommand(command.Id, args))
			}

		case "store":
			p.logger.Info("Receive DHT Store from %s", src.Id.String())
			if key, ok := command.Args["key"].(string); ok {
				if val, ok := command.Args["value"].(string); ok {
					p.kvs[key] = val
				}
			}

		case "find-value":
			p.logger.Info("Receive DHT Find-Node from %s", src.Id.String())
			if key, ok := command.Args["key"].(string); ok {
				args := map[string]interface{}{}
				if val, ok := p.kvs[key]; ok {
					args["value"] = val
				} else {
					hash := sha1.Sum([]byte(key))
					nodes := p.table.nearestNodes(NewNodeId(hash[:]))
					ary := make([]map[string]string, len(nodes))
					for i, n := range nodes {
						ary[i] = map[string]string{
							"id":   string(n.Id.Bytes()),
							"addr": n.Addr.String(),
						}
					}
					args["nodes"] = ary
				}
				p.sendPacket(src.Id, newRpcReturnCommand(command.Id, args))
			}

		case "": // callback
			id := string(command.Id)
			if ch := p.getRpcRetChan(id); ch != nil {
				ch <- &dhtRpcReturn{command: command, addr: addr}
			}
		}
	}
}

func (p *dht) close() {
	close(p.rpcChan)
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

func (p *dht) getRpcRetChan(id string) chan<- *dhtRpcReturn {
	if v, ok := p.rpcRet[id]; ok {
		delete(p.rpcRet, id)
		return v
	} else {
		return nil
	}
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

	//p.addRetChanRequest <- dhtRpcRetunChan{id: id, ch: ch}
	p.rpcRet[id] = ch
	p.rpcChan <- dhtPacket{Dst: dst, Payload: data}

	if len(command.Method) > 0 {
		/*
			go func() {
				// timeout
				<-time.After(20 * time.Millisecond)
				//if ch := p.getRpcRetChan(id); ch != nil {
				ch <- nil
				//	p.table.remove(dst)
				//	p.logger.Info("Remove %s from routing table", dst.String())
				//}
			}()
		*/
	}

	return ch
}
