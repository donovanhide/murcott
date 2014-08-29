package main

import (
	"bytes"
	"encoding/base64"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/h2so5/murcott"
	"github.com/martini-contrib/render"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"reflect"
	"strings"
)

func main() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(martini.Static("static"))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", "")
	})

	m.Get("/chat/:key", func(r render.Render, params martini.Params) {
		r.HTML(200, "chat", params["key"])
	})

	m.Get("/ws/:key", ws)

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

/*
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
*/

type JsonRpc struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type JsonRpcListener struct {
	client *murcott.Client
}

func (r JsonRpcListener) HandleLog(c func(args []interface{})) {
	go func() {
		for {
			var buf [1024]byte
			len, err := r.client.Logger.Read(buf[:])
			if err != nil {
				return
			}
			c([]interface{}{string(buf[:len])})
		}
	}()
}

func (r JsonRpcListener) HandleMessage(c func(args []interface{})) {
	r.client.HandleMessages(func(src murcott.NodeId, msg murcott.ChatMessage) {
		c([]interface{}{src.String(), msg.Text()})
	})
}

func (r JsonRpcListener) SendMessage(dst string, msg string, c func(args []interface{})) {
	id, err := murcott.NewNodeIdFromString(dst)
	if err == nil {
		r.client.SendMessage(id, murcott.NewPlainChatMessage(msg), func(ok bool) {
			c([]interface{}{ok})
		})
	} else {
		c([]interface{}{false})
	}
}

func (r JsonRpcListener) AddFriend(idstr string) {
	id, err := murcott.NewNodeIdFromString(idstr)
	if err == nil {
		r.client.Roster.Add(id)
	}
}

func (r JsonRpcListener) GetRoster(c func(args []interface{})) {
	list := make([]string, 0)
	for _, id := range r.client.Roster.List() {
		list = append(list, id.String())
	}
	c([]interface{}{list})
}

func (r JsonRpcListener) GetProfile(c func(args []interface{})) {
	profile := r.client.Profile()
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if profile.Avatar.Image != nil {
		png.Encode(encoder, profile.Avatar.Image)
	}
	c([]interface{}{profile.Nickname, string(buf.Bytes())})
}

func (r JsonRpcListener) SetProfile(nickname string, avatar string) {
	var profile murcott.UserProfile
	profile.Nickname = nickname
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(avatar))
	m, _, err := image.Decode(reader)
	if err == nil {
		profile.Avatar = murcott.UserAvatar{m}
	}
	r.client.SetProfile(profile)
}

func (r JsonRpcListener) SetStatus(status string) {
	if status == murcott.StatusActive {
		r.client.SetStatus(murcott.UserStatus{Type: murcott.StatusActive})
	} else if status == murcott.StatusAway {
		r.client.SetStatus(murcott.UserStatus{Type: murcott.StatusAway})
	}
}

func ws(w http.ResponseWriter, r *http.Request, params martini.Params) {

	var key *murcott.PrivateKey
	key = murcott.PrivateKeyFromString(params["key"])
	if key == nil {
		key = murcott.GeneratePrivateKey()
	}

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		return
	}

	storage := murcott.NewStorage(key.PublicKeyHash().String() + ".sqlite3")
	c := murcott.NewClient(key, storage)
	go c.Run()
	defer c.Close()

	v := reflect.ValueOf(JsonRpcListener{client: c})

	for {
		var rpc JsonRpc
		err := ws.ReadJSON(&rpc)
		if err != nil {
			break
		}

		f := v.MethodByName(rpc.Method)

		if f.IsValid() && f.Type().NumIn() == len(rpc.Params) {

			var args []reflect.Value
			for i, a := range rpc.Params {
				var val reflect.Value
				if f.Type().In(i).Kind() == reflect.Func {
					if id, ok := a.(float64); ok {
						val = reflect.ValueOf(func(args []interface{}) {
							retrpc := struct {
								Version string        `json:"jsonrpc"`
								Result  []interface{} `json:"result"`
								Id      int           `json:"id"`
							}{
								Version: "2.0",
								Result:  args,
								Id:      int(id),
							}
							ws.WriteJSON(retrpc)
						})
					} else {
						val = reflect.ValueOf(func(args []interface{}) {})
					}
				} else {
					val = reflect.ValueOf(a)
				}
				args = append(args, val)
			}

			f.Call(args)
		}
	}
}
