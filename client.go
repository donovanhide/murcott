package murcott

import (
	"github.com/vmihailenco/msgpack"
)

type Client struct {
	node *Node
}

func NewClient() *Client {
	return &Client{}
}

func (p *Client) SendMessage(dst NodeId, msg interface{}) {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		panic(err)
	}
	p.node.SendRawMessage(dst, data)
}
