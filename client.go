package murcott

import (
	"errors"
	"github.com/vmihailenco/msgpack"
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
	logger := NewLogger()
	node := newNode(key, logger)

	c := Client{
		node:   node,
		recv:   make(chan msgpair),
		exit:   make(chan struct{}),
		Logger: logger,
	}

	go node.run()
	go c.run()
	return &c
}

func (p *Client) run() {
	ch := p.node.messageChannel()

	for {
		select {
		case m := <-ch:
			var t struct {
				Type string `msgpack:"type"`
			}
			err := msgpack.Unmarshal(m.payload, &t)
			if err == nil {
				p.parseMessage(t.Type, m.payload, m.id)
			}
		case <-p.exit:
			return
		}
	}
}

func (p *Client) parseMessage(typ string, payload []byte, id NodeId) {
	switch typ {
	case "chat":
		chat := struct {
			Content ChatMessage `msgpack:"content"`
		}{}
		if msgpack.Unmarshal(payload, &chat) == nil {
			p.recv <- msgpair{id: id, msg: chat.Content}
		}
	default:
		p.Logger.Error("Unknown message type: %s", typ)
	}
}

func (p *Client) Send(dst NodeId, msg Message) error {
	var t struct {
		Type    string  `msgpack:"type"`
		Content Message `msgpack:"content"`
	}
	t.Content = msg

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

func (p *Client) Recv() (NodeId, Message) {
	m := <-p.recv
	return m.id, m.msg
}

func (p *Client) Close() {
	p.exit <- struct{}{}
	p.node.close()
}
