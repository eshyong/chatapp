package main

import (
	"log"
	"net/http"

	"github.com/eshyong/chatapp/chatserver"
	_ "github.com/lib/pq"
)

func main() {
	chatServer := chatserver.NewDefaultServer()
	http.Handle("/", chatServer.SetupRouter())

	addr := ":8080"
	log.Println("Starting server on " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err)
	}
}
