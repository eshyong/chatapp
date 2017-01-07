package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"fmt"
	"path/filepath"
)

const staticDir = "/static/"
var fileServerPath = filepath.Join(".", staticDir)

func main() {
	router := setupRoutes()
	http.Handle("/", router)

	addr := "localhost:8080"
	fmt.Println("Starting server on " + addr)
	http.ListenAndServe(addr, nil)
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(fileServerPath))))
	r.HandleFunc("/", serveHomePage)
	return r
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, filepath.Join(fileServerPath, "html", "index.html"))
}
