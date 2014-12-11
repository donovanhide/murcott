package main

import (
	"fmt"

	"github.com/h2so5/murcott"
	"github.com/h2so5/murcott/utils"
)

func main() {
	key := utils.GeneratePrivateKey()
	client, err := murcott.NewClient(key, utils.DefaultConfig)
	if err != nil {
		return
	}
	go func() {
		for {
			var buf [1024]byte
			len, err := client.Logger.Read(buf[:])
			if err != nil {
				return
			}
			fmt.Println(string(buf[:len]))
		}
	}()
	client.Run()
}
