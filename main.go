package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/eshyong/chatapp/chatserver"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	var user string
	err = db.QueryRow("SELECT current_user").Scan(&user)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(user)

	chatServer := chatserver.NewDefaultServer()
	http.Handle("/", chatServer.SetupRouter())

	addr := ":8080"
	log.Println("Starting server on " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err)
	}
}
