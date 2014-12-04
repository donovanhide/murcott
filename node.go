package murcott

import (
	"crypto/rand"
	"errors"
	"reflect"
	"time"

	"github.com/h2so5/murcott/log"
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

type node struct {
	router        *router
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

func newNode(key *utils.PrivateKey, logger *log.Logger, config utils.Config) (*node, error) {
	router, err := newRouter(key, logger, config)
	if err != nil {
		return nil, err
	}

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

	return n, nil
}

func (p *node) run() {
	msg := make(chan message)

	// Discover bootstrap nodes
	p.router.discover(p.config.Bootstrap())

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

func (p *node) parseMessage(typ string, payload []byte, id utils.NodeID) {
	if t, ok := p.name2type[typ]; ok {
		p.parseCommand(payload, id, t)
	} else {
		p.logger.Error("Unknown message type: %s", typ)
	}
}

func (p *node) parseCommand(payload []byte, id utils.NodeID, typ reflect.Type) {
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

func (p *node) send(dst utils.NodeID, msg interface{}, handler func(interface{})) error {
	return p.sendWithID(dst, msg, handler, "")
}

func (p *node) sendWithID(dst utils.NodeID, msg interface{}, handler func(interface{}), id string) error {
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
	packetID := p.router.sendMessage(dst, data)

	go func(msgID string, packetID int) {
		<-time.After(time.Second)
		p.cancelHandler <- msgID
		p.cancelMessage <- packetID
	}(t.ID, packetID)

	return nil
}

func (p *node) addNode(info utils.NodeInfo) {
	p.router.addNode(info)
}

func (p *node) knownNodes() []utils.NodeInfo {
	return p.router.knownNodes()
}

func (p *node) handle(handler func(utils.NodeID, interface{}) interface{}) {
	p.handler = handler
}

func (p *node) close() {
	p.router.close()
	p.exit <- struct{}{}
}
