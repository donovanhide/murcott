package main

import (
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/h2so5/murcott"
	"github.com/martini-contrib/render"
	"net/http"
)

func main() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(martini.Static("static"))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", "")
	})
	m.Get("/ws", ws)

	m.Get("/newkey", func(r render.Render) {
		key := murcott.GeneratePrivateKey()
		r.JSON(200, map[string]interface{}{
			"id":  key.PublicKeyHash().String(),
			"key": key.String(),
		})
	})

	m.Run()
}

type Message struct {
	MsgType string      `json:"type"`
	Payload interface{} `json:"payload"`
	Dst     string      `json:"dst"`
	Src     string      `json:"src"`
}

type Session struct {
	client *murcott.Client
	ws     *websocket.Conn
}

func (s *Session) WriteLog(msg string) {
	s.ws.WriteJSON(Message{
		MsgType: "log",
		Payload: struct {
			What string `json:"what"`
		}{msg},
	})
}

func (s *Session) WriteMessage(src murcott.NodeId, payload interface{}) {
	s.ws.WriteJSON(Message{
		MsgType: "msg",
		Src:     src.String(),
		Payload: payload,
	})
}

func (s *Session) WriteRoster(roster *murcott.Roster) {
	list := make([]string, 0)
	for _, id := range roster.List() {
		list = append(list, id.String())
	}
	s.ws.WriteJSON(Message{
		MsgType: "roster",
		Payload: list,
	})
}

func (s *Session) WriteProfile(id murcott.NodeId, profile murcott.UserProfile) {
	s.ws.WriteJSON(Message{
		MsgType: "profile",
		Src:     id.String(),
		Payload: map[string]interface{}{
			"nickname": profile.Nickname,
			"ext":      profile.Extension,
		},
	})
}

func ws(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	var key *murcott.PrivateKey
	if len(r.Form["key"]) == 1 {
		keystr := r.Form["key"][0]
		key = murcott.PrivateKeyFromString(keystr)
	}
	if key == nil {
		key = murcott.GeneratePrivateKey()
	}

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		return
	}

	storage := murcott.NewStorage(key.PublicKeyHash().String() + ".sqlite3")
	c := murcott.NewClient(key, storage)

	s := Session{
		client: c,
		ws:     ws,
	}

	go func() {
		for {
			var buf [1024]byte
			len, _ := c.Logger.Read(buf[:])
			s.WriteLog(string(buf[:len]))
		}
	}()

	c.HandleMessages(func(src murcott.NodeId, msg murcott.ChatMessage) {
		s.WriteMessage(src, msg.Text())
	})

	go c.Run()

	c.SetStatus(murcott.UserStatus{
		Type:    murcott.StatusActive,
		Message: ":-)",
	})

	s.WriteRoster(s.client.Roster)

	for _, id := range c.Roster.List() {
		c.RequestProfile(id, func(p *murcott.UserProfile) {
			if p != nil {
				s.WriteProfile(id, *p)
			}
		})
	}

	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		switch msg.MsgType {
		case "add-friend":
			if str, ok := msg.Payload.(string); ok {
				id, err := murcott.NewNodeIdFromString(str)
				if err == nil {
					s.client.Roster.Add(id)
					c.RequestProfile(id, func(p *murcott.UserProfile) {
						if p != nil {
							s.WriteProfile(id, *p)
						}
					})
				}
			}
		case "remove-friend":
			if str, ok := msg.Payload.(string); ok {
				id, err := murcott.NewNodeIdFromString(str)
				if err == nil {
					s.client.Roster.Remove(id)
				}
			}
		case "msg":
			id, err := murcott.NewNodeIdFromString(msg.Dst)
			if err == nil {
				if body, ok := msg.Payload.(string); ok {
					s.client.SendMessage(id, murcott.NewPlainChatMessage(body), func(ok bool) {
						if ok {
							s.WriteLog("ok")
						} else {
							s.WriteLog("timeout")
						}
					})
				}
			}
		}
	}

	c.Close()
}
