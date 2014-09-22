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

type node struct {
	router    *router
	handler   func(NodeId, interface{}) interface{}
	idmap     map[string]func(interface{})
	name2type map[string]reflect.Type
	type2name map[reflect.Type]string
	timeout   chan string
	logger    *Logger
	exit      chan struct{}
}

func newNode(key *PrivateKey, logger *Logger) *node {
	router := newRouter(key, logger)

	n := &node{
		router:    router,
		idmap:     make(map[string]func(interface{})),
		name2type: make(map[string]reflect.Type),
		type2name: make(map[reflect.Type]string),
		timeout:   make(chan string),
		logger:    logger,
		exit:      make(chan struct{}),
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

		case id := <-p.timeout:
			if h, ok := p.idmap[id]; ok {
				h(nil)
				delete(p.idmap, id)
			}
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
		p.idmap[t.Id] = handler
		go func() {
			<-time.After(time.Second)
			p.timeout <- t.Id
		}()
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
	p.router.sendMessage(dst, data)
	return nil
}

func (p *node) handle(handler func(NodeId, interface{}) interface{}) {
	p.handler = handler
}

func (p *node) close() {
	p.router.close()
	p.exit <- struct{}{}
}
