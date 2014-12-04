// Package murcott is a decentralized instant messaging framework.
package murcott

import (
	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/node"
	"github.com/h2so5/murcott/utils"
)

type Client struct {
	node          *node.Node
	msgHandler    messageHandler
	statusHandler statusHandler
	storage       *Storage
	status        UserStatus
	profile       UserProfile
	id            utils.NodeID
	Roster        *Roster
	BlockList     *BlockList
	Logger        *log.Logger
}

type messageHandler func(src utils.NodeID, msg ChatMessage)
type statusHandler func(src utils.NodeID, status UserStatus)

// NewClient generates a Client with the given PrivateKey.
func NewClient(key *utils.PrivateKey, storage *Storage, config utils.Config) (*Client, error) {
	logger := log.NewLogger()
	roster, _ := storage.LoadRoster()
	blocklist, _ := storage.LoadBlockList()

	node, err := node.NewNode(key, logger, config)
	if err != nil {
		return nil, err
	}

	node.RegisterMessageType("chat", ChatMessage{})
	node.RegisterMessageType("ack", messageAck{})
	node.RegisterMessageType("profile-req", userProfileRequest{})
	node.RegisterMessageType("profile-res", userProfileResponse{})
	node.RegisterMessageType("presence", userPresence{})

	knownNodes, _ := storage.loadKnownNodes()
	for _, n := range knownNodes {
		node.AddNode(n)
	}

	c := &Client{
		node:      node,
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

	c.node.Handle(func(src utils.NodeID, msg interface{}) interface{} {
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
				c.node.Send(src, userPresence{Status: c.status, Ack: true}, nil)
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
	status.Type = StatusOffline
	for _, n := range c.Roster.List() {
		c.node.Send(n, userPresence{Status: status, Ack: false}, nil)
	}

	c.storage.SaveRoster(c.Roster)
	c.storage.SaveBlockList(c.BlockList)
	c.storage.saveKnownNodes(c.node.KnownNodes())
	c.node.Close()
}

// Sends the given message to the destination node.
func (c *Client) SendMessage(dst utils.NodeID, msg ChatMessage, ack func(ok bool)) {
	c.node.Send(dst, msg, func(r interface{}) {
		ack(r != nil)
	})
}

// HandleMessages registers the given function as a massage handler.
func (c *Client) HandleMessages(handler func(src utils.NodeID, msg ChatMessage)) {
	c.msgHandler = handler
}

// HandleStatuses registers the given function as a status handler.
func (c *Client) HandleStatuses(handler func(src utils.NodeID, status UserStatus)) {
	c.statusHandler = handler
}

// Requests a user profile to the destination node.
// If no response is received from the node, RequestProfile tries to load a profile from the cache.
func (c *Client) RequestProfile(dst utils.NodeID, f func(profile *UserProfile)) {
	c.node.Send(dst, userProfileRequest{}, func(r interface{}) {
		if p, ok := r.(userProfileResponse); ok {
			c.storage.SaveProfile(dst, p.Profile)
			f(&p.Profile)
		} else {
			f(c.storage.LoadProfile(dst))
		}
	})
}

func (c *Client) ID() utils.NodeID {
	return c.id
}

func (c *Client) Status() UserStatus {
	return c.status
}

func (c *Client) SetStatus(status UserStatus) {
	c.status = status
	for _, n := range c.Roster.List() {
		c.node.Send(n, userPresence{Status: c.status, Ack: false}, nil)
	}
}

func (c *Client) Profile() UserProfile {
	return c.profile
}

func (c *Client) SetProfile(profile UserProfile) {
	c.storage.SaveProfile(c.id, profile)
	c.profile = profile
}
