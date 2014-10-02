package murcott

import (
	"crypto/rand"
	"errors"
	"reflect"
	"time"

	"github.com/vmihailenco/msgpack"
)

type msgpair struct {
	id  NodeId
	msg interface{}
}

type msghandler struct {
	id       string
	callback func(interface{})
}

type node struct {
	router        *router
	handler       func(NodeId, interface{}) interface{}
	idmap         map[string]func(interface{})
	name2type     map[string]reflect.Type
	type2name     map[reflect.Type]string
	register      chan msghandler
	cancelHandler chan string
	cancelMessage chan int
	config        Config
	logger        *Logger
	exit          chan struct{}
}

func newNode(key *PrivateKey, logger *Logger, config Config) *node {
	router := newRouter(key, logger, config)

	n := &node{
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

	n.registerMessageType("chat", ChatMessage{})
	n.registerMessageType("ack", messageAck{})
	n.registerMessageType("profile-req", userProfileRequest{})
	n.registerMessageType("profile-res", userProfileResponse{})
	n.registerMessageType("presence", userPresence{})

	return n
}

func (p *node) run() {
	msg := make(chan message)

	// Discover bootstrap nodes
	p.router.discover(p.config.getBootstrap())

	go func() {
		for {
			m, err := p.router.recvMessage()
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
			err := msgpack.Unmarshal(m.payload, &t)
			if err == nil {
				p.parseMessage(t.Type, m.payload, m.id)
			}

		case h := <-p.register:
			p.idmap[h.id] = h.callback

		case id := <-p.cancelHandler:
			if h, ok := p.idmap[id]; ok {
				h(nil)
				delete(p.idmap, id)
			}

		case id := <-p.cancelMessage:
			p.router.cancelMessage(id)

		case <-p.exit:
			return
		}
	}
}

func (p *node) registerMessageType(name string, typ interface{}) {
	t := reflect.TypeOf(typ)
	p.name2type[name] = t
	p.type2name[t] = name
}

func (p *node) parseMessage(typ string, payload []byte, id NodeId) {
	if t, ok := p.name2type[typ]; ok {
		p.parseCommand(payload, id, t)
	} else {
		p.logger.error("Unknown message type: %s", typ)
	}
}

func (p *node) parseCommand(payload []byte, id NodeId, typ reflect.Type) {
	c := struct {
		Content interface{} `msgpack:"content"`
		Id      string      `msgpack:"id"`
	}{}
	v := reflect.New(typ)
	c.Content = v.Interface()
	if msgpack.Unmarshal(payload, &c) == nil {
		if h, ok := p.idmap[c.Id]; ok {
			h(reflect.Indirect(v).Interface())
			delete(p.idmap, c.Id)
		} else if p.handler != nil {
			r := p.handler(id, reflect.Indirect(v).Interface())
			if r != nil {
				p.sendWithId(id, r, nil, c.Id)
			}
		}
	}
}

func (p *node) send(dst NodeId, msg interface{}, handler func(interface{})) error {
	return p.sendWithId(dst, msg, handler, "")
}

func (p *node) sendWithId(dst NodeId, msg interface{}, handler func(interface{}), id string) error {
	t := struct {
		Type    string      `msgpack:"type"`
		Content interface{} `msgpack:"content"`
		Id      string      `msgpack:"id"`
	}{Content: msg}

	if len(id) != 0 {
		t.Id = id
	} else if handler != nil {
		r := make([]byte, 10)
		rand.Read(r)
		t.Id = string(r)
		p.register <- msghandler{t.Id, handler}
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
	packetId := p.router.sendMessage(dst, data)

	go func(msgId string, packetId int) {
		<-time.After(time.Second)
		p.cancelHandler <- msgId
		p.cancelMessage <- packetId
	}(t.Id, packetId)

	return nil
}

func (p *node) addNode(info nodeInfo) {
	p.router.addNode(info)
}

func (p *node) knownNodes() []nodeInfo {
	return p.router.knownNodes()
}

func (p *node) handle(handler func(NodeId, interface{}) interface{}) {
	p.handler = handler
}

func (p *node) close() {
	p.router.close()
	p.exit <- struct{}{}
}
