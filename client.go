// Package murcott is a decentralized instant messaging framework.
package murcott

type Client struct {
	node       *node
	msgHandler messageHandler
	storage    *Storage
	status     UserStatus
	profile    UserProfile
	id         NodeId
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
		status:  UserStatus{Type: StatusOffline},
		id:      key.PublicKeyHash(),
		Roster:  roster,
		Logger:  logger,
	}

	profile := storage.loadProfile(c.id)
	if profile != nil {
		c.profile = *profile
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
		case userPresence:
			return userPresence{Status: c.status}
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
// If no response is received from the node, RequestProfile tries to load a profile from the cache.
func (c *Client) RequestProfile(dst NodeId, f func(profile *UserProfile)) {
	c.node.send(dst, userProfileRequest{}, func(r interface{}) {
		if p, ok := r.(userProfileResponse); ok {
			c.storage.saveProfile(dst, p.Profile)
			f(&p.Profile)
		} else {
			f(c.storage.loadProfile(dst))
		}
	})
}

// Requests a user status to the destination node.
// If no response is received from the node, RequestStatus returns an offline status.
func (c *Client) RequestStatus(dst NodeId, f func(profile UserStatus)) {
	c.node.send(dst, userPresence{Status: c.status}, func(r interface{}) {
		if p, ok := r.(userPresence); ok {
			f(p.Status)
		} else {
			f(UserStatus{Type: StatusOffline})
		}
	})
}

func (c *Client) Status() UserStatus {
	return c.status
}

func (c *Client) SetStatus(status UserStatus) {
	c.status = status
}

func (c *Client) Profile() UserProfile {
	return c.profile
}

func (c *Client) SetProfile(profile UserProfile) {
	c.storage.saveProfile(c.id, profile)
	c.profile = profile
}
