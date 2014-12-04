package node

import (
	"crypto/rand"
	"errors"
	"reflect"
	"time"

	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/router"
	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type msgpair struct {
	id  utils.NodeID
	msg interface{}
}

type msghandler struct {
	id       string
	callback func(interface{})
}

type Node struct {
	router        *router.Router
	handler       func(utils.NodeID, interface{}) interface{}
	idmap         map[string]func(interface{})
	name2type     map[string]reflect.Type
	type2name     map[reflect.Type]string
	register      chan msghandler
	cancelHandler chan string
	cancelMessage chan int
	config        utils.Config
	logger        *log.Logger
	exit          chan struct{}
}

func NewNode(key *utils.PrivateKey, logger *log.Logger, config utils.Config) (*Node, error) {
	router, err := router.NewRouter(key, logger, config)
	if err != nil {
		return nil, err
	}

	n := &Node{
		router:        router,
		idmap:         make(map[string]func(interface{})),
		name2type:     make(map[string]reflect.Type),
		type2name:     make(map[reflect.Type]string),
		register:      make(chan msghandler, 2),
		cancelHandler: make(chan string),
		cancelMessage: make(chan int),
		config:        config,
		logger:        logger,
		exit:          make(chan struct{}),
	}

	return n, nil
}

func (p *Node) Run() {
	msg := make(chan router.Message)

	// Discover bootstrap nodes
	p.router.Discover(p.config.Bootstrap())

	go func() {
		for {
			m, err := p.router.RecvMessage()
			if err != nil {
				break
			}
			msg <- m
		}
	}()

	for {
		select {
		case m := <-msg:
			var t struct {
				Type string `msgpack:"type"`
			}
			err := msgpack.Unmarshal(m.Payload, &t)
			if err == nil {
				p.parseMessage(t.Type, m.Payload, m.ID)
			}

		case h := <-p.register:
			p.idmap[h.id] = h.callback

		case id := <-p.cancelHandler:
			if h, ok := p.idmap[id]; ok {
				h(nil)
				delete(p.idmap, id)
			}

		case id := <-p.cancelMessage:
			p.router.CancelMessage(id)

		case <-p.exit:
			return
		}
	}
}

func (p *Node) RegisterMessageType(name string, typ interface{}) {
	t := reflect.TypeOf(typ)
	p.name2type[name] = t
	p.type2name[t] = name
}

func (p *Node) parseMessage(typ string, payload []byte, id utils.NodeID) {
	if t, ok := p.name2type[typ]; ok {
		p.parseCommand(payload, id, t)
	} else {
		p.logger.Error("Unknown message type: %s", typ)
	}
}

func (p *Node) parseCommand(payload []byte, id utils.NodeID, typ reflect.Type) {
	c := struct {
		Content interface{} `msgpack:"content"`
		ID      string      `msgpack:"id"`
	}{}
	v := reflect.New(typ)
	c.Content = v.Interface()
	if msgpack.Unmarshal(payload, &c) == nil {
		if h, ok := p.idmap[c.ID]; ok {
			h(reflect.Indirect(v).Interface())
			delete(p.idmap, c.ID)
		} else if p.handler != nil {
			r := p.handler(id, reflect.Indirect(v).Interface())
			if r != nil {
				p.sendWithID(id, r, nil, c.ID)
			}
		}
	}
}

func (p *Node) Send(dst utils.NodeID, msg interface{}, handler func(interface{})) error {
	return p.sendWithID(dst, msg, handler, "")
}

func (p *Node) sendWithID(dst utils.NodeID, msg interface{}, handler func(interface{}), id string) error {
	t := struct {
		Type    string      `msgpack:"type"`
		Content interface{} `msgpack:"content"`
		ID      string      `msgpack:"id"`
	}{Content: msg}

	if len(id) != 0 {
		t.ID = id
	} else if handler != nil {
		r := make([]byte, 10)
		rand.Read(r)
		t.ID = string(r)
		p.register <- msghandler{t.ID, handler}
	}

	if n, ok := p.type2name[reflect.TypeOf(msg)]; ok {
		t.Type = n
	} else {
		return errors.New("Unknown message type")
	}

	data, err := msgpack.Marshal(t)
	if err != nil {
		return err
	}
	packetID := p.router.SendMessage(dst, data)

	go func(msgID string, packetID int) {
		<-time.After(time.Second)
		p.cancelHandler <- msgID
		p.cancelMessage <- packetID
	}(t.ID, packetID)

	return nil
}

func (p *Node) AddNode(info utils.NodeInfo) {
	p.router.AddNode(info)
}

func (p *Node) KnownNodes() []utils.NodeInfo {
	return p.router.KnownNodes()
}

func (p *Node) Handle(handler func(utils.NodeID, interface{}) interface{}) {
	p.handler = handler
}

func (p *Node) Close() {
	p.router.Close()
	p.exit <- struct{}{}
}
