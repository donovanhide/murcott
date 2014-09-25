package main

import "github.com/h2so5/murcott"

func main() {
	key := murcott.GeneratePrivateKey()
	storage := murcott.NewStorage(":memory:")
	client := murcott.NewClient(key, storage, murcott.DefaultConfig)
	client.Run()
}
