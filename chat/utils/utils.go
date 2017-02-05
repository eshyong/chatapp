package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/lib/pq"
)

func UnmarshalJsonRequest(r *http.Request, model interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// Probably EOF errors, according to golang docs
		return err
	}

	if err := json.Unmarshal(body, model); err != nil {
		return err
	}
	return nil
}

func HandlePqError(w http.ResponseWriter, pqErr *pq.Error, uniqueMessage string) {
	log.Println(pqErr.Code.Name())
	switch pqErr.Code.Name() {
	case "unique_violation":
		http.Error(w, uniqueMessage, http.StatusBadRequest)
	default:
		http.Error(w, "Sorry, something went wrong. Please try again later.", http.StatusInternalServerError)
	}
}
