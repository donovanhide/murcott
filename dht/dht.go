package dht

import (
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type dhtRPCCallback func(*dhtRPCCommand, *net.UDPAddr)

type dhtRPCReturn struct {
	command dhtRPCCommand
	addr    net.Addr
}

type DHT struct {
	id    utils.NodeID
	table nodeTable
	k     int

	kvs      map[string]string
	kvsMutex sync.RWMutex

	chmap      map[string]chan<- dhtRPCReturn
	chmapMutex sync.Mutex

	conn net.PacketConn

	logger *log.Logger
}

type dhtRPCCommand struct {
	Src    utils.NodeID           `msgpack:"src"`
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

func NewDHT(k int, id utils.NodeID, conn net.PacketConn, logger *log.Logger) *DHT {
	d := DHT{
		id:     id,
		table:  newNodeTable(k, id),
		k:      k,
		kvs:    make(map[string]string),
		chmap:  make(map[string]chan<- dhtRPCReturn),
		conn:   conn,
		logger: logger,
	}
	return &d
}

func (p *DHT) ProcessPacket(b []byte, addr net.Addr) {
	var c dhtRPCCommand
	err := msgpack.Unmarshal(b, &c)
	if err != nil {
		p.logger.Error("%v", err)
		return
	}

	if !p.id.NS.Match(c.Src.NS) {
		return
	}

	if p.id.Digest.Cmp(c.Src.Digest) == 0 {
		return
	}

	p.table.insert(utils.NodeInfo{ID: c.Src, Addr: addr})

	switch c.Method {
	case "ping":
		p.logger.Info("%s: Receive DHT Ping from %s", p.id.String(), c.Src.String())
		p.sendPacket(c.Src, newRPCReturnCommand(c.ID, nil))

	case "find-node":
		p.logger.Info("%s: Receive DHT Find-Node from %s", p.id.String(), c.Src.String())
		if id, ok := c.Args["id"].(string); ok {
			args := map[string]interface{}{}
			nid, err := utils.NewNodeIDFromBytes([]byte(id))
			if err != nil {
				p.logger.Error("find-node: %v", err)
			} else {
				args["nodes"] = p.table.nearestNodes(nid)
				p.sendPacket(c.Src, newRPCReturnCommand(c.ID, args))
			}
		}

	case "store":
		p.logger.Info("%s: Receive DHT Store from %s", p.id.String(), c.Src.String())
		if key, ok := c.Args["key"].(string); ok {
			if val, ok := c.Args["value"].(string); ok {
				p.kvsMutex.Lock()
				p.kvs[key] = val
				p.kvsMutex.Unlock()
			}
		}

	case "find-value":
		p.logger.Info("%s: Receive DHT Find-Value from %s", p.id.String(), c.Src.String())
		if key, ok := c.Args["key"].(string); ok {
			args := map[string]interface{}{}
			p.kvsMutex.RLock()
			if val, ok := p.kvs[key]; ok {
				args["value"] = val
			} else {
				hash := sha1.Sum([]byte(key))
				n := p.table.nearestNodes(utils.NewNodeID(c.Src.NS, hash))
				args["nodes"] = n
			}
			p.kvsMutex.RUnlock()
			p.sendPacket(c.Src, newRPCReturnCommand(c.ID, args))
		}

	case "": // callback
		id := string(c.ID)
		p.chmapMutex.Lock()
		defer p.chmapMutex.Unlock()
		if ch, ok := p.chmap[id]; ok {
			delete(p.chmap, id)
			ch <- dhtRPCReturn{command: c, addr: addr}
		}
	}
}

func (p *DHT) FindNearestNode(findid utils.NodeID) []utils.NodeInfo {
	if !p.id.NS.Match(findid.NS) {
		return nil
	}

	reqch := make(chan utils.NodeInfo, 100)
	endch := make(chan struct{}, 100)

	f := func(id utils.NodeID, command dhtRPCCommand) {
		defer func() { endch <- struct{}{} }()
		ret, err := p.sendAndWaitPacket(id, command)
		if err == nil {
			if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []utils.NodeInfo
				ret.command.getArgs("nodes", &nodes)
				for _, n := range nodes {
					if n.ID.Digest.Cmp(p.id.Digest) != 0 {
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
			if _, ok := requested[node.ID.Digest.String()]; !ok {
				requested[node.ID.Digest.String()] = node
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

func (p *DHT) LoadValue(key string) *string {

	p.kvsMutex.RLock()
	if v, ok := p.kvs[key]; ok {
		p.kvsMutex.RUnlock()
		return &v
	}
	p.kvsMutex.RUnlock()

	hash := sha1.Sum([]byte(key))
	keyid := utils.NewNodeID(p.id.NS, hash)

	retch := make(chan *string, 2)
	reqch := make(chan utils.NodeID, 100)
	endch := make(chan struct{}, 100)

	nodes := p.table.nearestNodes(keyid)

	f := func(id utils.NodeID, keyid utils.NodeID, command dhtRPCCommand) {
		ret, err := p.sendAndWaitPacket(id, command)
		if err == nil {
			if val, ok := ret.command.Args["value"].(string); ok {
				retch <- &val
			} else if _, ok := ret.command.Args["nodes"]; ok {
				var nodes []utils.NodeInfo
				ret.command.getArgs("nodes", &nodes)
				dist := id.Digest.Xor(keyid.Digest)
				for _, n := range nodes {
					p.table.insert(n)
					if dist.Cmp(n.ID.Digest.Xor(keyid.Digest)) == 1 {
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
			if _, ok := requested[id.Digest.String()]; !ok {
				requested[id.Digest.String()] = struct{}{}
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

func (p *DHT) StoreValue(key string, value string) {
	hash := sha1.Sum([]byte(key))
	c := newRPCCommand("store", map[string]interface{}{
		"key":   key,
		"value": value,
	})
	for _, n := range p.FindNearestNode(utils.NewNodeID(p.id.NS, hash)) {
		p.sendPacket(n.ID, c)
	}
}

func (p *DHT) AddNode(node utils.NodeInfo) {
	if !p.id.NS.Match(node.ID.NS) {
		return
	}
	p.table.insert(node)
	p.sendPing(node.ID)
}

func (p *DHT) KnownNodes() []utils.NodeInfo {
	return p.table.nodes()
}

func (p *DHT) GetNodeInfo(id utils.NodeID) *utils.NodeInfo {
	return p.table.find(id)
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

func (p *DHT) Discover(addr net.Addr) error {
	c := newRPCCommand("ping", nil)
	c.Src = p.id
	b, err := msgpack.Marshal(c)
	if err != nil {
		return err
	}
	_, err = p.conn.WriteTo(b, addr)
	if err != nil {
		return err
	}
	return nil
}

func (p *DHT) sendPing(dst utils.NodeID) error {
	c := newRPCCommand("ping", nil)
	return p.sendPacket(dst, c)
}

func (p *DHT) sendPacket(dst utils.NodeID, c dhtRPCCommand) error {
	c.Src = p.id
	i := p.GetNodeInfo(dst)
	if i == nil || i.Addr == nil {
		return errors.New("route not found")
	}
	b, err := msgpack.Marshal(c)
	if err != nil {
		return err
	}
	_, err = p.conn.WriteTo(b, i.Addr)
	if err != nil {
		return err
	}
	return nil
}

func (p *DHT) sendAndWaitPacket(dst utils.NodeID, c dhtRPCCommand) (dhtRPCReturn, error) {
	ch := make(chan dhtRPCReturn, 2)

	p.chmapMutex.Lock()
	p.chmap[string(c.ID)] = ch
	p.chmapMutex.Unlock()

	defer func() {
		p.chmapMutex.Lock()
		delete(p.chmap, string(c.ID))
		p.chmapMutex.Unlock()
	}()

	p.sendPacket(dst, c)
	select {
	case r := <-ch:
		return r, nil
	case <-time.After(time.Second):
		return dhtRPCReturn{}, errors.New("timeout")
	}
}

func (p *DHT) Close() error {
	return p.conn.Close()
}
