package murcott

import (
	"github.com/vmihailenco/msgpack"
)

type Message struct {
	Src     NodeId
	Payload interface{}
}

type Client struct {
	node    *node
	logger  *Logger
	msgChan chan Message
	exit    chan struct{}
}

func NewClient() *Client {
	logger := NewLogger()
	node := newNode(logger)
	ch := node.messageChannel()
	go node.run()

	c := Client{
		node:    node,
		logger:  logger,
		msgChan: make(chan Message),
		exit:    make(chan struct{}),
	}

	go func() {
		for {
			select {
			case msg := <-ch:
				var out map[string]interface{}
				err := msgpack.Unmarshal(msg.payload, &out)
				if err == nil {
					c.msgChan <- Message{Src: msg.id, Payload: out}
				}
			case <-c.exit:
				return
			}
		}
	}()

	return &c
}

func (p *Client) Logger() *Logger {
	return p.logger
}

func (p *Client) Send(dst NodeId, msg interface{}) {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		panic(err)
	}
	p.node.sendMessage(dst, data)
}

func (p *Client) Recv() Message {
	return <-p.msgChan
}

func (p *Client) Close() {
	p.exit <- struct{}{}
	p.node.close()
}
