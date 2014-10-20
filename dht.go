package murcott

import (
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack"
)

type dhtRPCCallback func(*dhtRPCCommand, *net.UDPAddr)

type dhtPacket struct {
	dst     NodeID
	payload []byte
}

type dhtRPCReturn struct {
	command dhtRPCCommand
	addr    *net.UDPAddr
}

type rpcReturnMap struct {
	chmap map[string]chan *dhtRPCReturn
	mutex *sync.Mutex
}

func (p *rpcReturnMap) push(id string, c chan *dhtRPCReturn) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.chmap[id] = c
}

func (p *rpcReturnMap) pop(id string) chan *dhtRPCReturn {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if c, ok := p.chmap[id]; ok {
		delete(p.chmap, id)
		return c
	} else {
		return nil
	}
}

type keyValueStore struct {
	storage map[string]string
	mutex   *sync.Mutex
}

func (p *keyValueStore) set(key, value string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.storage[key] = value
}

func (p *keyValueStore) get(key string) (string, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if v, ok := p.storage[key]; ok {
		return v, true
	} else {
		return "", false
	}
}

type dht struct {
	info   nodeInfo
	table  nodeTable
	k      int
	kvs    keyValueStore
	rpcRet rpcReturnMap
	rpc    chan dhtPacket
	logger *Logger
}

type dhtRPCCommand struct {
	ID     []byte                 `msgpack:"id"`
	Method string                 `msgpack:"method"`
	Args   map[string]interface{} `msgpack:"args"`
}

func (p *dhtRPCCommand) getArgs(k string, v ...interface{}) {
	b, err := msgpack.Marshal(p.Args[k])
	if err == nil {
		msgpack.Unmarshal(b, v...)
	}
}

func newDht(k int, info nodeInfo, logger *Logger) *dht {
	d := dht{
		info:  info,
		table: newNodeTable(k, info.ID),
		k:     k,
		kvs: keyValueStore{
			storage: make(map[string]string),
			mutex:   &sync.Mutex{},
		},
		rpcRet: rpcReturnMap{
			chmap: make(map[string]chan *dhtRPCReturn),
			mutex: &sync.Mutex{},
		},
		rpc:    make(chan dhtPacket, 100),
		logger: logger,
	}
	return &d
}

func (p *dht) addNode(node nodeInfo) {
	p.table.insert(node)
	p.sendPing(node.ID)
}

func (p *dht) knownNodes() []nodeInfo {
	return p.table.nodes()
}

func (p *dht) getNodeInfo(id NodeID) *nodeInfo {
	return p.table.find(id)
}

func (p *dht) storeValue(key string, value string) {
	hash := sha1.Sum([]byte(key))
	c := newRPCCommand("store", map[string]interface{}{
		"key":   key,
		"value": value,
	})
	for _, n := range p.findNearestNode(NewNodeID(hash)) {
		p.sendPacket(n.ID, c)
	}
}

func (p *dht) findNearestNode(findid NodeID) []nodeInfo {

	reqch := make(chan nodeInfo, 100)
	endch := make(chan struct{}, 100)

	f := func(id NodeID, command dhtRPCCommand) {
		ch := p.sendPacket(id, command)

		ret := <-ch
		if ret != nil {
			if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []nodeInfo
				ret.command.getArgs("nodes", &nodes)
				for _, n := range nodes {
					if n.ID.cmp(p.info.ID) != 0 {
						p.table.insert(n)
						reqch <- n
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
			if _, ok := requested[node.ID.String()]; !ok {
				requested[node.ID.String()] = node
				c := newRPCCommand("find-node", map[string]interface{}{
					"id": string(findid.Bytes()),
				})
				go f(node.ID, c)
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

	if v, ok := p.kvs.get(key); ok {
		return &v
	}

	hash := sha1.Sum([]byte(key))
	keyid := NewNodeID(hash)

	retch := make(chan *string, 2)
	reqch := make(chan NodeID, 100)
	endch := make(chan struct{}, 100)

	nodes := p.table.nearestNodes(NewNodeID(hash))

	f := func(id NodeID, keyid NodeID, command dhtRPCCommand) {
		ch := p.sendPacket(id, command)

		ret := <-ch
		if ret != nil {
			if val, ok := ret.command.Args["value"].(string); ok {
				retch <- &val
			} else if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []nodeInfo
				ret.command.getArgs("nodes", &nodes)
				dist := id.xor(keyid)
				for _, n := range nodes {
					p.table.insert(n)
					if dist.cmp(n.ID.xor(keyid)) == 1 {
						reqch <- n.ID
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
		reqch <- n.ID
	}

	count := 0
	requested := make(map[string]struct{})

	for {
		select {
		case id := <-reqch:
			if _, ok := requested[id.String()]; !ok {
				requested[id.String()] = struct{}{}
				c := newRPCCommand("find-value", map[string]interface{}{
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

func (p *dht) processPacket(src nodeInfo, payload []byte) {
	var command dhtRPCCommand
	err := msgpack.Unmarshal(payload, &command)
	if err == nil {
		p.table.insert(src)

		switch command.Method {
		case "ping":
			p.logger.info("Receive DHT Ping from %s", src.ID.String())
			p.sendPacket(src.ID, newRPCReturnCommand(command.ID, nil))

		case "find-node":
			p.logger.info("Receive DHT Find-Node from %s", src.ID.String())
			if id, ok := command.Args["id"].(string); ok {
				args := map[string]interface{}{}
				var idary [20]byte
				copy(idary[:], []byte(id)[:20])
				args["nodes"] = p.table.nearestNodes(NewNodeID(idary))
				p.sendPacket(src.ID, newRPCReturnCommand(command.ID, args))
			}

		case "store":
			p.logger.info("Receive DHT Store from %s", src.ID.String())
			if key, ok := command.Args["key"].(string); ok {
				if val, ok := command.Args["value"].(string); ok {
					p.kvs.set(key, val)
				}
			}

		case "find-value":
			p.logger.info("Receive DHT Find-Node from %s", src.ID.String())
			if key, ok := command.Args["key"].(string); ok {
				args := map[string]interface{}{}
				if val, ok := p.kvs.get(key); ok {
					args["value"] = val
				} else {
					hash := sha1.Sum([]byte(key))
					args["nodes"] = p.table.nearestNodes(NewNodeID(hash))
				}
				p.sendPacket(src.ID, newRPCReturnCommand(command.ID, args))
			}

		case "": // callback
			id := string(command.ID)
			if ch := p.rpcRet.pop(id); ch != nil {
				ch <- &dhtRPCReturn{command: command, addr: src.Addr}
			}
		}
	}
}

func (p *dht) nextPacket() (NodeID, []byte, error) {
	if c, ok := <-p.rpc; ok {
		return c.dst, c.payload, nil
	} else {
		return NodeID{}, nil, errors.New("DHT closed")
	}
}

func newRPCCommand(method string, args map[string]interface{}) dhtRPCCommand {
	id := make([]byte, 20)
	_, err := rand.Read(id)
	if err != nil {
		panic(err)
	}
	return dhtRPCCommand{
		ID:     id,
		Method: method,
		Args:   args,
	}
}

func newRPCReturnCommand(id []byte, args map[string]interface{}) dhtRPCCommand {
	return dhtRPCCommand{
		ID:     id,
		Method: "",
		Args:   args,
	}
}

func (p *dht) sendPing(dst NodeID) {
	c := newRPCCommand("ping", nil)
	ch := p.sendPacket(dst, c)
	go func() {
		ret := <-ch
		if ret != nil {
			p.logger.info("Receive DHT ping response")
		}
	}()
}

func (p *dht) sendPacket(dst NodeID, command dhtRPCCommand) chan *dhtRPCReturn {
	data, err := msgpack.Marshal(command)
	if err != nil {
		panic(err)
	}

	id := string(command.ID)
	ch := make(chan *dhtRPCReturn, 2)

	p.rpcRet.push(id, ch)
	p.rpc <- dhtPacket{dst: dst, payload: data}
	go func(id string) {
		<-time.After(time.Second)
		if c := p.rpcRet.pop(id); c != nil {
			c <- nil
		}
	}(id)
	return ch
}

func (p *dht) close() {
	close(p.rpc)
}
