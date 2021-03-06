package main

import (
	"time"

	"example.com/card-server/pkg"
)

func main() {
	server := pkg.NewServer()
	server.RegisterHandler(pkg.NewQueueManager())
	server.RegisterHandler(pkg.NewGameManager(30 * time.Second))
	server.RegisterHandler(pkg.NewMatchManager(30 * time.Second))

	server.Listen("0.0.0.0:8080")
}
