// +build !race

package murcott

import (
	"encoding/base64"
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/h2so5/murcott/client"
	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/node"
	"github.com/h2so5/murcott/utils"
)

func TestClientMessage(t *testing.T) {
	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	client1, err := NewClient(key1, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	client2, err := NewClient(key2, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	success := make(chan bool)
	plainmsg := client.NewPlainChatMessage("Hello")

	client2.HandleMessages(func(src utils.NodeID, msg client.ChatMessage) {
		if src.Cmp(key1.PublicKeyHash()) == 0 {
			if msg.Text() == plainmsg.Text() {
				success <- true
			} else {
				t.Errorf("wrong message body")
				success <- false
			}
		} else {
			t.Errorf("wrong source id")
			success <- false
		}
	})

	client1.SendMessage(key2.PublicKeyHash(), plainmsg, func(ok bool) {
		if ok {
			success <- true
		} else {
			t.Errorf("SendMessage() timed out")
			success <- false
		}
	})

	go client1.Run()
	go client2.Run()

	for i := 0; i < 2; i++ {
		if !<-success {
			return
		}
	}

	client1.Close()
	client2.Close()
}

func TestClientBlockList(t *testing.T) {
	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	client1, err := NewClient(key1, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	client2, err := NewClient(key2, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	success := make(chan bool)
	plainmsg := client.NewPlainChatMessage("Hello")

	client2.BlockList.Add(key1.PublicKeyHash())

	client1.SendMessage(key2.PublicKeyHash(), plainmsg, func(ok bool) {
		if ok {
			t.Errorf("BlockList does not work")
			success <- false
		} else {
			success <- true
		}
	})

	go client1.Run()
	go client2.Run()

	if !<-success {
		return
	}

	client1.Close()
	client2.Close()
}

func TestClientProfile(t *testing.T) {
	const data = `iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAIAAAD8GO2jAAADw0lEQVRIx+1WXU
xTZxjuhTe7UTMSMy6mA5wttJRz+p2/7/yXnpYON3SAP9MRiBFZwtgWxcRE1Bm3sC0jcz+GLLohm8sM
m1Nh6gLqyMAoVXRQKFBByk8pLQWENep2tVe2W7eWo1db8uUkJznned7vfd7nfV/DGsr2RI/hv0dgpG
zptC2LZzI52rTw+ngI/sbFDLaLqiYTmSaCsCCBNdG6CQAaUAjMyA5JwcielpxvWXmgcseb5cUkmQn3
0EsA6IhnsmX2JUtKVdkrDfUf3bzVPDJxa3/lDtKakcWzegmsHC1L7Gsbc6+0nxmb7IrM+e/+PnK55V
styaA4Fb0pgvzwdlF5bkXDV4dnHwQmpn1wHoZfttkhUCTP6hU5g0GqQyoSrN3eS+E5/1jYG40NNp+r
z01NgvBBdl0E8DMSOcVmrqmqCE37gtO+0Gx/IHhz77bCHIXlVQHbJUrkLAwy0WgxBJBfSZPdKUkXL5
yAwMci3qnYUHNjXTE2b1jnstNW9JRBQRZJ5c3MoggsLKUq+I1C1+Bwx+RsP4Q/PHqjZournEl9fa1Y
XVnadPqLr+sP06Y0SsLGRAngB1YRVNOq458disz7x6d6wvO3rzR9+V31rraLJ339v4DakbsD71VVYN
JsE7mECdJppGhyoTXF0/Hj1G+DwWjvRLQ3MNYJV4FEwTMaGzrbUOtYZlAX1E4sRRAOwTMqR75d/upo
qCsY9QFBaKYvEhuCRIEY4bkBuESZi3NqkpWjjFTiBHy2JCUvbfrhKNgqMn8b/DUe7u7ruXxnuAM4Jm
f6jry7WyNMWBX+2WuGR9QPYiT8Ik9e85zv7vn5QmPd8U8Pfryz5MhWzT/QNnM/0Npyct3zKxwuNYNG
xkX4wAiNk6VynFJxgTvftsa56un1OKuEWHnq2AfT9+4MjXj2lby81sHr6qbgYYKjEIukbAnUXp/nKr
Wjrq6WmQeBU7XvbF1iyHMr0AQXT/BXlzazlBXcoMk5ltRP9ldMxQZ/vdb4+fa8b46+X1qUT1AEfKC3
FwkyLnCKezY4Oz3noF79/vZQtPcSaCAhTsbpj/ZwXASUzFueTT5WWx2MeKFMxyM9oHBb6/cFacsddp
7AtN5uSgock2n68MBbkwvoYK6r7WeK0OoXXHI8Cv87AfgZBNhIpHZ2/jT3x+h1z/ntKunWRGii8aDH
NXA4VWCfWX6irsbray11sm47TGccJ3pcI9PMULzAbMrTtuSqDpHmFtCNj3EvAtPBWEaYonkoKTY9Ef
R496KHSxGDMljK+P9u+gTOnzzpHBoBGFBEAAAAAElFTkSuQmCC`

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	client1, err := NewClient(key1, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	client2, err := NewClient(key2, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	img, _, err := image.Decode(reader)
	if err != nil {
		t.Errorf("fail to decode image: %v", err)
		return
	}

	profile := client.UserProfile{
		Nickname: "nick",
		Avatar:   client.UserAvatar{Image: img},
		Extension: map[string]string{
			"Location": "Tokyo",
		},
	}
	client2.SetProfile(profile)

	success := make(chan bool)

	go client1.Run()
	go client2.Run()

	client1.RequestProfile(key2.PublicKeyHash(), func(p *client.UserProfile) {
		if p == nil {
			t.Errorf("UserProfile should not be nil")
			success <- false
		} else if p.Nickname != profile.Nickname {
			t.Errorf("wrong Nickname: %s; expects %s", p.Nickname, profile.Nickname)
			success <- false
		} else if !reflect.DeepEqual(p.Extension, profile.Extension) {
			t.Errorf("wrong Extension: %v; expects %v", p.Extension, profile.Extension)
			success <- false
		} else {
		loop:
			for x := 0; x < p.Avatar.Image.Bounds().Max.X; x++ {
				for y := 0; y < p.Avatar.Image.Bounds().Max.Y; y++ {
					r1, g1, b1, a1 := p.Avatar.Image.At(x, y).RGBA()
					r2, g2, b2, a2 := profile.Avatar.Image.At(x, y).RGBA()
					if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
						t.Errorf("avatar image color mismatch at (%d, %d)", x, y)
						success <- false
						break loop
					}
				}
			}
			success <- true
		}
	})

	if !<-success {
		return
	}

	client1.Close()
	client2.Close()
}

func TestClientStatus(t *testing.T) {
	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	client1, err := NewClient(key1, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	client2, err := NewClient(key2, client.NewStorage(":memory:"), utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	status1 := client.UserStatus{Type: client.StatusActive, Message: ":-("}

	client1.Roster.Add(key2.PublicKeyHash())
	client2.Roster.Add(key1.PublicKeyHash())

	success := make(chan bool)

	client1.HandleStatuses(func(src utils.NodeID, p client.UserStatus) {
		if p.Type != client.StatusOffline {
			t.Errorf("wrong Type: %s; expects %s", p.Type, client.StatusOffline)
			success <- false
			return
		}
		if p.Message != "" {
			t.Errorf("wrong Message: %s; expects %s", p.Message, "")
			success <- false
			return
		}
		success <- true
	})

	client2.HandleStatuses(func(src utils.NodeID, p client.UserStatus) {
		if p.Type != status1.Type {
			t.Errorf("wrong Type: %s; expects %s", p.Type, status1.Type)
			success <- false
			return
		}
		if p.Message != status1.Message {
			t.Errorf("wrong Message: %s; expects %s", p.Message, status1.Message)
			success <- false
			return
		}
		success <- true
	})

	go client1.Run()
	go client2.Run()

	client1.SetStatus(status1)

	for i := 0; i < 2; i++ {
		if !<-success {
			return
		}
	}

	client1.Close()
	client2.Close()
}

func TestNodeChatMessage(t *testing.T) {
	logger := log.NewLogger()
	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	node1, err := node.NewNode(key1, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	node2, err := node.NewNode(key2, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	plainmsg := client.NewPlainChatMessage("Hello")

	success := make(chan bool)

	node2.Handle(func(src utils.NodeID, msg interface{}) interface{} {
		if m, ok := msg.(client.ChatMessage); ok {
			if m.Text() == plainmsg.Text() {
				if src.Cmp(key1.PublicKeyHash()) == 0 {
					success <- true
				} else {
					t.Errorf("wrong source id")
					success <- false
				}
			} else {
				t.Errorf("wrong message body")
				success <- false
			}
		} else {
			t.Errorf("wrong message type")
			success <- false
		}
		return client.MessageAck{}
	})

	node1.Send(key2.PublicKeyHash(), plainmsg, func(msg interface{}) {
		if _, ok := msg.(client.MessageAck); ok {
			success <- true
		} else {
			t.Errorf("wrong ack type")
			success <- false
		}
	})

	go node1.Run()
	go node2.Run()

	for i := 0; i < 2; i++ {
		if !<-success {
			return
		}
	}

	node1.Close()
	node2.Close()
}
