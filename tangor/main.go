package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2so5/murcott"
	"github.com/h2so5/murcott/client"
	"github.com/h2so5/murcott/utils"
	"github.com/wsxiaoys/terminal/color"
)

func main() {
	path := os.Getenv("TANGORPATH")
	if path == "" {
		path = os.Getenv("HOME") + "/.tangor"
	}

	keyfile := flag.String("i", path+"/id_dsa", "Identity file")
	flag.Parse()

	fmt.Println()
	color.Print("@{Gk} @{Yk}  tangor  @{Gk} @{|}\n")
	fmt.Println()

	key, err := getKey(*keyfile)
	if err != nil {
		color.Printf(" -> @{Rk}ERROR:@{|} %v\n", err)
		os.Exit(-1)
	}

	id := utils.NewNodeID([4]byte{1, 1, 1, 1}, key.Digest())
	color.Printf("Your ID: @{Wk} %s @{|}\n\n", id.String())

	client, err := murcott.NewClient(key, utils.DefaultConfig)
	if err != nil {
		panic(err)
	}

	s := Session{cli: client}
	s.bootstrap()
	s.commandLoop()
}

func getKey(keyfile string) (*utils.PrivateKey, error) {
	_, err := os.Stat(filepath.Dir(keyfile))

	if _, err := os.Stat(keyfile); err != nil {
		err := os.MkdirAll(filepath.Dir(keyfile), 0755)
		if err != nil {
			return nil, err
		}
		key := utils.GeneratePrivateKey()
		pem, err := key.MarshalText()
		err = ioutil.WriteFile(keyfile, pem, 0644)
		if err != nil {
			return nil, err
		}
		fmt.Printf(" -> Create a new private key: %s\n", keyfile)
	}

	pem, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	var key utils.PrivateKey
	err = key.UnmarshalText(pem)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

type Session struct {
	cli *murcott.Client
}

func (s *Session) bootstrap() {
	fmt.Println(" -> Searching bootstrap nodes")
	go func() {
		s.cli.Run()
	}()

	var nodes int
	for i := 0; i < 5; i++ {
		nodes = s.cli.Nodes()
		fmt.Printf(" -> Found %d nodes", nodes)
		time.Sleep(200 * time.Millisecond)
		fmt.Printf("\r")
	}
	fmt.Println()

	if nodes == 0 {
		color.Printf(" -> @{Yk}WARNING:@{|} node not found\n")
	}
	fmt.Println()
}

func (s *Session) commandLoop() {
	var chatID *utils.NodeID

	s.cli.HandleMessages(func(src utils.NodeID, msg client.ChatMessage) {
		color.Printf("\r* @{Wk}%s@{|} %s\n", src.String()[:6], msg.Text())
	})

	bio := bufio.NewReader(os.Stdin)
	for {
		if chatID == nil {
			fmt.Print("> ")
		} else {
			fmt.Print("* ")
		}
		line, _, err := bio.ReadLine()
		if err != nil {
			return
		}
		c := strings.Split(string(line), " ")
		if len(c) == 0 || c[0] == "" {
			continue
		}
		switch c[0] {
		case "/chat":
			if len(c) != 2 {
				color.Printf(" -> @{Rk}ERROR:@{|} /chat takes 1 argument\n")
			} else {
				nid, err := utils.NewNodeIDFromString(c[1])
				if err != nil {
					color.Printf(" -> @{Rk}ERROR:@{|} invalid ID\n")
				} else {
					chatID = &nid
					color.Printf(" -> Start a chat with @{Wk} %s @{|}\n\n", nid.String())
				}
			}
		case "/end":
			if chatID != nil {
				color.Printf(" -> End current chat\n")
				chatID = nil
			}
		case "/exit", "/quit":
			color.Printf(" -> See you@{Kg}.@{Kr}.@{Ky}.@{|}\n")
			return
		case "/help":
			showHelp()
		default:
			if chatID == nil {
				color.Printf(" -> @{Rk}ERROR:@{|} unknown command\n")
				showHelp()
			} else {
				s.cli.SendMessage(*chatID, client.NewPlainChatMessage(string(line)), func(bool) {})
			}
		}
	}
}

func showHelp() {
	fmt.Println()
	color.Printf("  * HELP *\n")
	color.Printf("  @{Kg}/chat [ID]@{|}\tStart a chat with [ID]\n")
	color.Printf("  @{Kg}/end      @{|}\tEnd current chat\n")
	color.Printf("  @{Kg}/help     @{|}\tShow this message\n")
	color.Printf("  @{Kg}/exit     @{|}\tExit this program\n")
	fmt.Println()
}
