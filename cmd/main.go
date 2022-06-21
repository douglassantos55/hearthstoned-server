package main

import (
	"log"
	"net/http"
	"time"

	"example.com/card-server/pkg"
)

func main() {
	server := pkg.NewServer()
	server.RegisterHandler(pkg.NewGameManager())
	server.RegisterHandler(pkg.NewQueueManager())
	server.RegisterHandler(pkg.NewMatchManager(30 * time.Second))

	http.HandleFunc("/", server.HandleConnection)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
