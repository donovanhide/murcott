package murcott

import (
	"errors"
	"github.com/vmihailenco/msgpack"
	"reflect"
)

type Message interface{}

type ChatMessage struct {
	Body string `msgpack:"body"`
}

type msgpair struct {
	id  NodeId
	msg Message
}

type Client struct {
	node   *node
	recv   chan msgpair
	exit   chan struct{}
	Logger *Logger
}

func NewClient(key *PrivateKey) *Client {
	logger := newLogger()
	node := newNode(key, logger)

	c := Client{
		node:   node,
		recv:   make(chan msgpair),
		Logger: logger,
	}

	go c.run()
	return &c
}

func (p *Client) run() {
	for {
		id, payload, err := p.node.recvMessage()
		if err != nil {
			break
		}
		var t struct {
			Type string `msgpack:"type"`
		}
		err = msgpack.Unmarshal(payload, &t)
		if err == nil {
			p.parseMessage(t.Type, payload, id)
		}
	}
}

func (p *Client) parseMessage(typ string, payload []byte, id NodeId) {
	switch typ {
	case "chat":
		p.parseCommand(payload, id, ChatMessage{})
	default:
		p.Logger.error("Unknown message type: %s", typ)
	}
}

func (p *Client) parseCommand(payload []byte, id NodeId, typ interface{}) {
	c := struct {
		Content interface{} `msgpack:"content"`
	}{}
	v := reflect.New(reflect.ValueOf(typ).Type())
	c.Content = v.Interface()
	if msgpack.Unmarshal(payload, &c) == nil {
		p.recv <- msgpair{id: id, msg: reflect.Indirect(v).Interface()}
	}
}

func (p *Client) Send(dst NodeId, msg Message) error {
	t := struct {
		Type    string  `msgpack:"type"`
		Content Message `msgpack:"content"`
	}{Content: msg}

	switch msg.(type) {
	case ChatMessage:
		t.Type = "chat"
	default:
		return errors.New("Unknown message type")
	}

	data, err := msgpack.Marshal(t)
	if err != nil {
		return err
	}
	p.node.sendMessage(dst, data)
	return nil
}

func (p *Client) Recv() (NodeId, Message, error) {
	if m, ok := <-p.recv; ok {
		return m.id, m.msg, nil
	} else {
		return NodeId{}, nil, errors.New("Client closed")
	}
}

func (p *Client) Close() {
	close(p.recv)
	p.node.close()
}
