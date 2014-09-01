// Package murcott is a decentralized instant messaging framework.
package murcott

type Client struct {
	node          *node
	msgHandler    messageHandler
	statusHandler statusHandler
	storage       *Storage
	status        UserStatus
	profile       UserProfile
	id            NodeId
	Roster        *Roster
	BlockList     *BlockList
	Logger        *Logger
}

type messageHandler func(src NodeId, msg ChatMessage)
type statusHandler func(src NodeId, status UserStatus)

// NewClient generates a Client with the given PrivateKey.
func NewClient(key *PrivateKey, storage *Storage) *Client {
	logger := newLogger()
	roster, _ := storage.LoadRoster()
	blocklist, _ := storage.LoadBlockList()

	c := &Client{
		node:      newNode(key, logger),
		storage:   storage,
		status:    UserStatus{Type: StatusOffline},
		id:        key.PublicKeyHash(),
		Roster:    roster,
		BlockList: blocklist,
		Logger:    logger,
	}

	profile := storage.LoadProfile(c.id)
	if profile != nil {
		c.profile = *profile
	}

	c.node.handle(func(src NodeId, msg interface{}) interface{} {
		if c.BlockList.contains(src) {
			return nil
		}
		switch msg.(type) {
		case ChatMessage:
			if c.msgHandler != nil {
				c.msgHandler(src, msg.(ChatMessage))
			}
			return messageAck{}
		case userProfileRequest:
			return userProfileResponse{Profile: c.profile}
		case userPresence:
			p := msg.(userPresence)
			if c.statusHandler != nil {
				c.statusHandler(src, p.Status)
			}
			if !p.Ack {
				c.node.send(src, userPresence{Status: c.status, Ack: true}, nil)
			}
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
	status := c.status
	status.Type = StatusOffline
	for _, n := range c.Roster.List() {
		c.node.send(n, userPresence{Status: status, Ack: false}, nil)
	}

	c.storage.SaveRoster(c.Roster)
	c.storage.SaveBlockList(c.BlockList)
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

// HandleMessages registers the given function as a status handler.
func (c *Client) HandleStatuses(handler func(src NodeId, status UserStatus)) {
	c.statusHandler = handler
}

// Requests a user profile to the destination node.
// If no response is received from the node, RequestProfile tries to load a profile from the cache.
func (c *Client) RequestProfile(dst NodeId, f func(profile *UserProfile)) {
	c.node.send(dst, userProfileRequest{}, func(r interface{}) {
		if p, ok := r.(userProfileResponse); ok {
			c.storage.SaveProfile(dst, p.Profile)
			f(&p.Profile)
		} else {
			f(c.storage.LoadProfile(dst))
		}
	})
}

func (c *Client) Id() NodeId {
	return c.id
}

func (c *Client) Status() UserStatus {
	return c.status
}

func (c *Client) SetStatus(status UserStatus) {
	c.status = status
	for _, n := range c.Roster.List() {
		c.node.send(n, userPresence{Status: c.status, Ack: false}, nil)
	}
}

func (c *Client) Profile() UserProfile {
	return c.profile
}

func (c *Client) SetProfile(profile UserProfile) {
	c.storage.SaveProfile(c.id, profile)
	c.profile = profile
}
