package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"fmt"
	"path/filepath"
)

const staticDir = "/static/"

func main() {
	router := setupRoutes()
	http.Handle("/", router)

	addr := "localhost:8080"
	fmt.Println("Starting server on " + addr)
	http.ListenAndServe(addr, nil)
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	fileServerPath := filepath.Join(".", staticDir)
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(fileServerPath))))
	r.HandleFunc("/", serveHomePage)
	return r
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./public/html/index.html")
}
