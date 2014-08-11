package main

import (
	"fmt"
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
}

type Session struct {
	client *murcott.Client
	ws     *websocket.Conn
}

func (s *Session) Run() {
	s.WriteLog("DEBUG", "websocket connected")
	for {
		_, p, err := s.ws.ReadMessage()
		if err != nil {
			return
		}
		fmt.Println(p)
	}
}

func (s *Session) WriteLog(level string, msg string) {
	s.ws.WriteJSON(Message{
		MsgType: "log",
		Payload: struct {
			Level string `json:"level"`
			What  string `json:"what"`
		}{level, msg},
	})
}

func ws(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		return
	}

	s := Session{
		client: murcott.NewClient(),
		ws:     ws,
	}

	s.Run()
}
