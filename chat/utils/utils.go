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

func HandlePqError(w http.ResponseWriter, pqErr *pq.Error, message string) {
	log.Println(pqErr.Code.Name())
	if pqErr.Code.Name() == "unique_violation" {
		http.Error(w, message, http.StatusBadRequest)
	}
}
