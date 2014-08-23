// Package murcott is a decentralized instant messaging framework.
package murcott

type Client struct {
	node       *node
	msgHandler messageHandler
	Logger     *Logger
}

type messageHandler func(src NodeId, msg ChatMessage)

// NewClient generates a Client with the given PrivateKey.
func NewClient(key *PrivateKey) *Client {
	logger := newLogger()

	c := &Client{
		node:   newNode(key, logger),
		Logger: logger,
	}

	c.node.handle(func(src NodeId, msg interface{}) interface{} {
		switch msg.(type) {
		case ChatMessage:
			if c.msgHandler != nil {
				c.msgHandler(src, msg.(ChatMessage))
			}
			return messageAck{}
		}
		return nil
	})

	return c
}

// Start a mainloop in the current goroutine.
func (c *Client) Run() {
	c.node.run()
}

// Stop the current mainloop.
func (c *Client) Close() {
	c.node.close()
}

// Send the given message to the destination node.
func (c *Client) SendMessage(dst NodeId, msg ChatMessage, ack func(ok bool)) {
	c.node.send(dst, msg, func(r interface{}) {
		ack(r != nil)
	})
}

// HandleMessages registers the given function as a massage handler.
func (c *Client) HandleMessages(handler func(src NodeId, msg ChatMessage)) {
	c.msgHandler = handler
}
