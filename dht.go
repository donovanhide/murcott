package murcott

import (
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type dhtRPCCallback func(*dhtRPCCommand, *net.UDPAddr)

type dhtPacket struct {
	dst     utils.NodeID
	payload []byte
}

type dhtRPCReturn struct {
	command dhtRPCCommand
	addr    *net.UDPAddr
}

type dhtOutgoingPacket struct {
	dst      utils.NodeID
	command  dhtRPCCommand
	callback chan<- dhtRPCReturn
}

type dht struct {
	info  utils.NodeInfo
	table nodeTable
	k     int

	kvs      map[string]string
	kvsMutex sync.RWMutex

	chmap  map[string]chan<- dhtRPCReturn
	rpc    chan dhtPacket
	recvch chan packet
	sendch chan dhtOutgoingPacket

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

func newDht(k int, info utils.NodeInfo, logger *Logger) *dht {
	d := dht{
		info:   info,
		table:  newNodeTable(k, info.ID),
		k:      k,
		kvs:    make(map[string]string),
		chmap:  make(map[string]chan<- dhtRPCReturn),
		rpc:    make(chan dhtPacket, 100),
		recvch: make(chan packet, 100),
		sendch: make(chan dhtOutgoingPacket, 100),
		logger: logger,
	}
	go d.loop()
	return &d
}

func (p *dht) loop() {
	for {
		select {
		case pac := <-p.recvch:
			p.processPacket(pac)
		case pac := <-p.sendch:
			data, err := msgpack.Marshal(pac.command)
			if err == nil {
				p.rpc <- dhtPacket{dst: pac.dst, payload: data}
				if pac.callback != nil {
					p.chmap[string(pac.command.ID)] = pac.callback
				}
			}
		}
	}
}

func (p *dht) ProcessPacket(pac packet) {
	p.recvch <- pac
}

func (p *dht) processPacket(pac packet) {
	var command dhtRPCCommand
	err := msgpack.Unmarshal(pac.Payload, &command)
	if err == nil {
		p.table.insert(utils.NodeInfo{ID: pac.Src, Addr: pac.addr})

		switch command.Method {
		case "ping":
			p.logger.info("Receive DHT Ping from %s", pac.Src.String())
			p.sendPacket(pac.Src, newRPCReturnCommand(command.ID, nil))

		case "find-node":
			p.logger.info("Receive DHT Find-Node from %s", pac.Src.String())
			if id, ok := command.Args["id"].(string); ok {
				args := map[string]interface{}{}
				var idary [20]byte
				copy(idary[:], []byte(id)[:20])
				args["nodes"] = p.table.nearestNodes(utils.NewNodeID(idary))
				p.sendPacket(pac.Src, newRPCReturnCommand(command.ID, args))
			}

		case "store":
			p.logger.info("Receive DHT Store from %s", pac.Src.String())
			if key, ok := command.Args["key"].(string); ok {
				if val, ok := command.Args["value"].(string); ok {
					p.kvsMutex.Lock()
					p.kvs[key] = val
					p.kvsMutex.Unlock()
				}
			}

		case "find-value":
			p.logger.info("Receive DHT Find-Node from %s", pac.Src.String())
			if key, ok := command.Args["key"].(string); ok {
				args := map[string]interface{}{}
				p.kvsMutex.RLock()
				if val, ok := p.kvs[key]; ok {
					args["value"] = val
				} else {
					hash := sha1.Sum([]byte(key))
					args["nodes"] = p.table.nearestNodes(utils.NewNodeID(hash))
				}
				p.kvsMutex.RUnlock()
				p.sendPacket(pac.Src, newRPCReturnCommand(command.ID, args))
			}

		case "": // callback
			id := string(command.ID)
			if ch, ok := p.chmap[id]; ok {
				delete(p.chmap, id)
				ch <- dhtRPCReturn{command: command, addr: pac.addr}
			}
		}
	}
}

func (p *dht) addNode(node utils.NodeInfo) {
	p.table.insert(node)
	p.sendPing(node.ID)
}

func (p *dht) knownNodes() []utils.NodeInfo {
	return p.table.nodes()
}

func (p *dht) getNodeInfo(id utils.NodeID) *utils.NodeInfo {
	return p.table.find(id)
}

func (p *dht) storeValue(key string, value string) {
	hash := sha1.Sum([]byte(key))
	c := newRPCCommand("store", map[string]interface{}{
		"key":   key,
		"value": value,
	})
	for _, n := range p.findNearestNode(utils.NewNodeID(hash)) {
		p.sendPacket(n.ID, c)
	}
}

func (p *dht) findNearestNode(findid utils.NodeID) []utils.NodeInfo {

	reqch := make(chan utils.NodeInfo, 100)
	endch := make(chan struct{}, 100)

	f := func(id utils.NodeID, command dhtRPCCommand) {
		defer func() { endch <- struct{}{} }()
		ret := p.sendRecvPacket(id, command)
		if ret != nil {
			if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []utils.NodeInfo
				ret.command.getArgs("nodes", &nodes)
				for _, n := range nodes {
					if n.ID.Cmp(p.info.ID) != 0 {
						p.table.insert(n)
						reqch <- n
					}
				}
			}
		}
	}

	var res []utils.NodeInfo
	nodes := p.table.nearestNodes(findid)

	if len(nodes) == 0 {
		return res
	}

	for _, n := range nodes {
		reqch <- n
	}

	count := 0
	requested := make(map[string]utils.NodeInfo)

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

	sorter := utils.NodeInfoSorter{Nodes: res, ID: findid}
	sort.Sort(sorter)

	if len(sorter.Nodes) > p.k {
		return sorter.Nodes[:p.k]
	}
	return sorter.Nodes
}

func (p *dht) loadValue(key string) *string {

	p.kvsMutex.RLock()
	if v, ok := p.kvs[key]; ok {
		p.kvsMutex.RUnlock()
		return &v
	}
	p.kvsMutex.RUnlock()

	hash := sha1.Sum([]byte(key))
	keyid := utils.NewNodeID(hash)

	retch := make(chan *string, 2)
	reqch := make(chan utils.NodeID, 100)
	endch := make(chan struct{}, 100)

	nodes := p.table.nearestNodes(utils.NewNodeID(hash))

	f := func(id utils.NodeID, keyid utils.NodeID, command dhtRPCCommand) {
		ret := p.sendRecvPacket(id, command)
		if ret != nil {
			if val, ok := ret.command.Args["value"].(string); ok {
				retch <- &val
			} else if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []utils.NodeInfo
				ret.command.getArgs("nodes", &nodes)
				dist := id.Xor(keyid)
				for _, n := range nodes {
					p.table.insert(n)
					if dist.Cmp(n.ID.Xor(keyid)) == 1 {
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

func (p *dht) nextPacket() (utils.NodeID, []byte, error) {
	if c, ok := <-p.rpc; ok {
		return c.dst, c.payload, nil
	}
	return utils.NodeID{}, nil, errors.New("DHT closed")
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

func (p *dht) sendPing(dst utils.NodeID) {
	c := newRPCCommand("ping", nil)
	p.sendPacket(dst, c)
}

func (p *dht) sendPacket(dst utils.NodeID, command dhtRPCCommand) {
	p.sendch <- dhtOutgoingPacket{dst: dst, command: command, callback: nil}
}

func (p *dht) sendRecvPacket(dst utils.NodeID, command dhtRPCCommand) *dhtRPCReturn {
	ch := make(chan dhtRPCReturn, 2)
	p.sendch <- dhtOutgoingPacket{dst: dst, command: command, callback: ch}

	select {
	case r := <-ch:
		return &r
	case <-time.After(time.Second):
		return nil
	}
}

func (p *dht) close() {
	//close(p.rpc)
}
