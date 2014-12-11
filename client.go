// Package murcott is a decentralized instant messaging framework.
package murcott

import (
	"github.com/h2so5/murcott/client"
	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/node"
	"github.com/h2so5/murcott/utils"
)

type Client struct {
	node          *node.Node
	msgHandler    messageHandler
	statusHandler statusHandler
	status        client.UserStatus
	profile       client.UserProfile
	id            utils.NodeID
	Roster        *client.Roster
	Logger        *log.Logger
}

type messageHandler func(src utils.NodeID, msg client.ChatMessage)
type statusHandler func(src utils.NodeID, status client.UserStatus)

// NewClient generates a Client with the given PrivateKey.
func NewClient(key *utils.PrivateKey, config utils.Config) (*Client, error) {
	logger := log.NewLogger()

	node, err := node.NewNode(key, logger, config)
	if err != nil {
		return nil, err
	}

	node.RegisterMessageType("chat", client.ChatMessage{})
	node.RegisterMessageType("ack", client.MessageAck{})
	node.RegisterMessageType("profile-req", client.UserProfileRequest{})
	node.RegisterMessageType("profile-res", client.UserProfileResponse{})
	node.RegisterMessageType("presence", client.UserPresence{})

	c := &Client{
		node:   node,
		status: client.UserStatus{Type: client.StatusOffline},
		id:     utils.NewNodeID([4]byte{1, 1, 1, 1}, key.Digest()),
		Roster: &client.Roster{},
		Logger: logger,
	}

	c.node.Handle(func(src utils.NodeID, msg interface{}) interface{} {
		switch msg.(type) {
		case client.ChatMessage:
			if c.msgHandler != nil {
				c.msgHandler(src, msg.(client.ChatMessage))
			}
			return client.MessageAck{}
		case client.UserProfileRequest:
			return client.UserProfileResponse{Profile: c.profile}
		case client.UserPresence:
			p := msg.(client.UserPresence)
			if c.statusHandler != nil {
				c.statusHandler(src, p.Status)
			}
			if !p.Ack {
				c.node.Send(src, client.UserPresence{Status: c.status, Ack: true}, nil)
			}
		}
		return nil
	})

	return c, nil
}

// Starts a mainloop in the current goroutine.
func (c *Client) Run() {
	c.node.Run()
}

// Stops the current mainloop.
func (c *Client) Close() {
	status := c.status
	status.Type = client.StatusOffline
	for _, n := range c.Roster.List() {
		c.node.Send(n, client.UserPresence{Status: status, Ack: false}, nil)
	}

	c.node.Close()
}

// Sends the given message to the destination node.
func (c *Client) SendMessage(dst utils.NodeID, msg client.ChatMessage, ack func(ok bool)) {
	c.node.Send(dst, msg, func(r interface{}) {
		ack(r != nil)
	})
}

// HandleMessages registers the given function as a massage handler.
func (c *Client) HandleMessages(handler func(src utils.NodeID, msg client.ChatMessage)) {
	c.msgHandler = handler
}

// HandleStatuses registers the given function as a status handler.
func (c *Client) HandleStatuses(handler func(src utils.NodeID, status client.UserStatus)) {
	c.statusHandler = handler
}

// Requests a user profile to the destination node.
// If no response is received from the node, RequestProfile tries to load a profile from the cache.
func (c *Client) RequestProfile(dst utils.NodeID, f func(profile *client.UserProfile)) {
	c.node.Send(dst, client.UserProfileRequest{}, func(r interface{}) {
		if p, ok := r.(client.UserProfileResponse); ok {
			f(&p.Profile)
		} else {
			f(nil)
		}
	})
}

func (c *Client) ID() utils.NodeID {
	return c.id
}

func (c *Client) Status() client.UserStatus {
	return c.status
}

func (c *Client) SetStatus(status client.UserStatus) {
	c.status = status
	for _, n := range c.Roster.List() {
		c.node.Send(n, client.UserPresence{Status: c.status, Ack: false}, nil)
	}
}

func (c *Client) Profile() client.UserProfile {
	return c.profile
}

func (c *Client) SetProfile(profile client.UserProfile) {
	c.profile = profile
}
