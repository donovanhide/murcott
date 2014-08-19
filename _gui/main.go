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
	logger *murcott.Logger
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

	c := murcott.NewClient(key)
	logger := c.Logger()
	logger.Info("websocket connected")

	s := Session{
		client: c,
		ws:     ws,
		logger: logger,
	}

	logch := logger.Channel()
	exit := make(chan struct{})

	go func() {
		for {
			select {
			case log := <-logch:
				s.WriteLog(log)
			case <-exit:
				return
			}
		}
	}()

	go func() {
		for {
			msg := c.Recv()
			s.WriteMessage(msg.Src, msg.Payload)
		}
	}()

	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		switch msg.MsgType {
		case "msg":
			id, err := murcott.NewNodeIdFromString(msg.Dst)
			if err == nil {
				s.client.Send(id, msg.Payload)
			}
		}
	}

	exit <- struct{}{}
	s.client.Close()
}
