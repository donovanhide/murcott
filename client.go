package murcott

import (
	"github.com/vmihailenco/msgpack"
)

type Client struct {
	node   *node
	logger *Logger
}

func NewClient() *Client {
	logger := NewLogger()
	return &Client{
		newNode(logger),
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
	p.node.sendMessage(dst, data)
}

func (p *Client) MessageCallback(cb func(NodeId, interface{})) {
	p.node.messageCallback(func(src NodeId, payload []byte) {
		var out map[string]interface{}
		err := msgpack.Unmarshal(payload, &out)
		if err == nil {
			cb(src, out)
		}
	})
}

func (p *Client) Close() {
	p.node.close()
}
