package main

import "github.com/h2so5/murcott"

func main() {
	key := murcott.GeneratePrivateKey()
	storage := murcott.NewStorage(":memory:")
	client, err := murcott.NewClient(key, storage, murcott.DefaultConfig)
	if err != nil {
		return
	}
	client.Run()
}
