package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/h2so5/murcott"
	"github.com/h2so5/murcott/utils"
	"github.com/wsxiaoys/terminal/color"
)

func main() {
	keyfile := flag.String("i", os.Getenv("HOME")+"/.tangor/id_dsa", "Identity file")
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

	fmt.Println(" -> Searching nodes")
	go func() {
		client.Run()
	}()

	var nodes int
	for i := 0; i < 5; i++ {
		nodes = client.Nodes()
		fmt.Printf(" -> Found %d nodes", nodes)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("\r")
	}
	fmt.Println()

	if nodes == 0 {
		color.Printf(" -> @{Yk}WARNING:@{|} node not found\n")
	}
	fmt.Println()

	var i int
	fmt.Scanf("%d", &i)
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
