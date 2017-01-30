package chat

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/eshyong/chatapp/chat/structs"
	"github.com/stretchr/testify/assert"
)

func TestReadUserCreds(t *testing.T) {
	body := []byte(`{"username": "eric", "password": "abc123"}`)
	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(body))),
	}
	expected := &structs.UserCreds{
		UserName: "eric",
		Password: "abc123",
	}
	actual, err := readUserCreds(req)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)

	body = []byte(`{"not": "a_valid_user"}`)
	req = &http.Request{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(body))),
	}
	u, err := readUserCreds(req)
	assert.Nil(t, u)
	assert.Error(t, err)
}
