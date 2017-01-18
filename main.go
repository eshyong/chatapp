package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eshyong/chatapp/chatserver"
	_ "github.com/lib/pq"
)

func main() {

	// TLS config taken from Filippo Valsorda's blog post:
	// https://blog.cloudflare.com/exposing-go-on-the-internet/
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	chatServer := chatserver.NewDefaultServer()
	addr := ":8443"
	if os.Getenv("CHATAPP_ADDR") != "" {
		addr = os.Getenv("CHATAPP_ADDR")
	}
	if os.Getenv("CHATAPP_TLS_PRIVATE_KEY") == "" || os.Getenv("CHATAPP_TLS_CERTIFICATE") == "" {
		log.Fatal("Need to set CHATAPP_TLS_PRIVATE_KEY and CHATAPP_TLS_CERTIFICATE to continue")
	}
	certFile := os.Getenv("CHATAPP_TLS_CERTIFICATE")
	keyFile := os.Getenv("CHATAPP_TLS_PRIVATE_KEY")

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig,
		Handler:      chatServer.SetupRouter(),
	}

	log.Println("Starting server on " + server.Addr)
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatalln(err)
	}
}
