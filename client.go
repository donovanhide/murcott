package murcott

import (
	"github.com/vmihailenco/msgpack"
)

type Client struct {
	node   *Node
	logger *Logger
}

func NewClient() *Client {
	logger := NewLogger()
	return &Client{
		NewNode(logger),
		logger,
	}
}

func (p *Client) Logger() *Logger {
	return p.logger
}

func (p *Client) SendMessage(dst NodeId, msg interface{}) {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		panic(err)
	}
	p.node.SendMessage(dst, data)
}

func (p *Client) MessageCallback(cb func(NodeId, interface{})) {
	p.node.MessageCallback(func(src NodeId, payload []byte) {
		var out map[string]interface{}
		err := msgpack.Unmarshal(payload, &out)
		if err == nil {
			cb(src, out)
		}
	})
}

func (p *Client) Close() {
	p.node.Close()
}
