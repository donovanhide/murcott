// Package murcott is a decentralized instant messaging framework.
package murcott

type Client struct {
	node       *node
	msgHandler messageHandler
	storage    *Storage
	profile    UserProfile
	Roster     *Roster
	Logger     *Logger
}

type messageHandler func(src NodeId, msg ChatMessage)

// NewClient generates a Client with the given PrivateKey.
func NewClient(key *PrivateKey, storage *Storage) *Client {
	logger := newLogger()
	roster, _ := storage.loadRoster()

	c := &Client{
		node:    newNode(key, logger),
		storage: storage,
		Roster:  roster,
		Logger:  logger,
	}

	c.node.handle(func(src NodeId, msg interface{}) interface{} {
		switch msg.(type) {
		case ChatMessage:
			if c.msgHandler != nil {
				c.msgHandler(src, msg.(ChatMessage))
			}
			return messageAck{}
		case userProfileRequest:
			return userProfileResponse{Profile: c.profile}
		}
		return nil
	})

	return c
}

// Starts a mainloop in the current goroutine.
func (c *Client) Run() {
	c.node.run()
}

// Stops the current mainloop.
func (c *Client) Close() {
	c.storage.saveRoster(c.Roster)
	c.node.close()
}

// Sends the given message to the destination node.
func (c *Client) SendMessage(dst NodeId, msg ChatMessage, ack func(ok bool)) {
	c.node.send(dst, msg, func(r interface{}) {
		ack(r != nil)
	})
}

// HandleMessages registers the given function as a massage handler.
func (c *Client) HandleMessages(handler func(src NodeId, msg ChatMessage)) {
	c.msgHandler = handler
}

// Requests a user profile to the destination node.
func (c *Client) RequestProfile(dst NodeId, f func(profile UserProfile)) {
	c.node.send(dst, userProfileRequest{}, func(r interface{}) {
		if p, ok := r.(userProfileResponse); ok {
			f(p.Profile)
		}
	})
}

func (c *Client) Profile() UserProfile {
	return c.profile
}

func (c *Client) SetProfile(profile UserProfile) {
	c.profile = profile
}
