all:
	go install github.com/eshyong/chatapp

fetchdeps:
	go get github.com/eshyong/...
