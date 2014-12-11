package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/h2so5/murcott"
	"github.com/h2so5/murcott/client"
	"github.com/h2so5/murcott/utils"
	"github.com/martini-contrib/render"
	"github.com/nfnt/resize"
)

func main() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(martini.Static("static"))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", "")
	})

	m.Get("/chat/:id/:key", func(r render.Render, params martini.Params) {
		j, _ := json.Marshal(params)
		r.HTML(200, "chat", string(j))
	})

	m.Get("/ws/:key", ws)

	m.Get("/newkey", func(r render.Render) {
		key := utils.GeneratePrivateKey()
		r.JSON(200, map[string]interface{}{
			"id":  key.Digest().String(),
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

type JsonRPC struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type JsonRPCListener struct {
	client *murcott.Client
}

func (r JsonRPCListener) HandleLog(c func(args []interface{})) {
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

func (r JsonRPCListener) HandleMessage(c func(args []interface{})) {
	r.client.HandleMessages(func(src utils.NodeID, msg client.ChatMessage) {
		c([]interface{}{src.String(), msg.Text(), msg.Time})
	})
}

func (r JsonRPCListener) HandleStatus(c func(args []interface{})) {
	r.client.HandleStatuses(func(src utils.NodeID, status client.UserStatus) {
		c([]interface{}{src.String(), status.Type})
	})
}

func (r JsonRPCListener) SendMessage(dst string, msg string, c func(args []interface{})) {
	id, err := utils.NewNodeIDFromString(dst)
	if err == nil {
		r.client.SendMessage(id, client.NewPlainChatMessage(msg), func(ok bool) {
			c([]interface{}{ok})
		})
	} else {
		c([]interface{}{false})
	}
}

func (r JsonRPCListener) AddFriend(idstr string) {
	id, err := utils.NewNodeIDFromString(idstr)
	if err == nil {
		r.client.Roster.Add(id)
	}
}

func (r JsonRPCListener) GetRoster(c func(args []interface{})) {
	list := make([]string, 0)
	for _, id := range r.client.Roster.List() {
		list = append(list, id.String())
	}
	c([]interface{}{list})
}

func (r JsonRPCListener) GetProfile(c func(args []interface{})) {
	profile := r.client.Profile()
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if profile.Avatar.Image != nil {
		png.Encode(encoder, profile.Avatar.Image)
	}
	c([]interface{}{r.client.ID().String(), profile.Nickname, string(buf.Bytes())})
}

func (r JsonRPCListener) GetFriendProfile(idstr string, c func(args []interface{})) {
	id, err := utils.NewNodeIDFromString(idstr)
	if err != nil {
		return
	}
	r.client.RequestProfile(id, func(profile *client.UserProfile) {
		if profile != nil {
			var buf bytes.Buffer
			encoder := base64.NewEncoder(base64.StdEncoding, &buf)
			if profile.Avatar.Image != nil {
				png.Encode(encoder, profile.Avatar.Image)
			}
			c([]interface{}{profile.Nickname, string(buf.Bytes())})
		}
	})
}

func (r JsonRPCListener) SetProfile(nickname string, avatar string, c func(args []interface{})) {
	profile := r.client.Profile()
	profile.Nickname = nickname
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(avatar))
	m, _, err := image.Decode(reader)
	if err == nil {
		m = resize.Thumbnail(32, 32, m, resize.Lanczos3)
		profile.Avatar = client.UserAvatar{m}
	}
	r.client.SetProfile(profile)
	c([]interface{}{})
}

func (r JsonRPCListener) SetStatus(status string) {
	if status == client.StatusActive {
		r.client.SetStatus(client.UserStatus{Type: client.StatusActive})
	} else if status == client.StatusAway {
		r.client.SetStatus(client.UserStatus{Type: client.StatusAway})
	}
}

func ws(w http.ResponseWriter, r *http.Request, params martini.Params) {

	var key *utils.PrivateKey
	key = utils.PrivateKeyFromString(params["key"])
	if key == nil {
		key = utils.GeneratePrivateKey()
	}

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		return
	}

	c, err := murcott.NewClient(key, utils.DefaultConfig)
	if err != nil {
		return
	}

	go c.Run()
	defer c.Close()

	v := reflect.ValueOf(JsonRPCListener{client: c})

	for {
		var rpc JsonRPC
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
								ID      int           `json:"id"`
							}{
								Version: "2.0",
								Result:  args,
								ID:      int(id),
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
