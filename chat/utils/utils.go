package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
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
