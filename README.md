murcott
=======

Decentralized instant messaging framework

[![Build Status](https://travis-ci.org/h2so5/murcott.svg)](https://travis-ci.org/h2so5/murcott)
[![Coverage Status](https://img.shields.io/coveralls/h2so5/murcott.svg)](https://coveralls.io/r/h2so5/murcott)
[![GoDoc](https://godoc.org/github.com/h2so5/murcott?status.svg)](http://godoc.org/github.com/h2so5/murcott)

## Installation

```
go get github.com/h2so5/murcott
```

## Example

```go
package main

import (
	"fmt"
	"github.com/h2so5/murcott"
	"os"
	"strings"
)

func main() {
	// Private key identifies the ownership of your node.
	key := murcott.GeneratePrivateKey()
	fmt.Println("Your node id: " + key.PublicKeyHash().String())

	// Storage keeps client's persistent data.
	storage := murcott.NewStorage("storage.sqlite3")

	// Create a client with the private key and the storage.
	client := murcott.NewClient(key, storage, murcott.DefaultConfig)

	// Handle incoming messages.
	client.HandleMessages(func(src murcott.NodeId, msg murcott.ChatMessage) {
		fmt.Println(msg.Text() + " from " + src.String())
	})

	// Start client's mainloop.
	go client.Run()

	// Parse a base58-encoded node identifier of your friend.
	dst, _ := murcott.NewNodeIdFromString("3CjjdZLV4DqXkc3KtPZPTfBU1AAY")

	for {
		b := make([]byte, 1024)
		len, err := os.Stdin.Read(b)
		if err != nil {
			break
		}
		str := strings.TrimSpace(string(b[:len]))
		if str == "quit" {
			break
		}

		// Send message to the destination node.
		client.SendMessage(dst, murcott.NewPlainChatMessage(str), func(ok bool) {
			if !ok {
				fmt.Println("Failed to deliver the message to the node...")
			}
		})
	}

	// Stop client's mainloop.
	client.Close()
}
```

## License

MIT License
