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

func (s *Session) Run() {
	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		switch msg.MsgType {
		case "msg":
			s.client.SendMessage(*murcott.NewNodeIdFromString(msg.Dst), msg.Payload)
		}
	}

	s.client.Close()
}

func (s *Session) WriteLog(msg string) {
	s.ws.WriteJSON(Message{
		MsgType: "log",
		Payload: struct {
			What string `json:"what"`
		}{msg},
	})
}

func ws(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		return
	}

	c := murcott.NewClient()
	logger := c.Logger()
	logger.Info("websocket connected")

	s := Session{
		client: c,
		ws:     ws,
		logger: logger,
	}

	go func(ch <-chan string) {
		for {
			msg := <-ch
			s.WriteLog(msg)
		}
	}(logger.Channel())

	c.MessageCallback(func(src murcott.NodeId, payload interface{}) {
		s.ws.WriteJSON(Message{
			MsgType: "msg",
			Src:     src.String(),
			Payload: payload,
		})
	})

	s.Run()
}
