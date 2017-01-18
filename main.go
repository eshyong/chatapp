package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
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
	httpsPort := "8443"
	httpPort := "8080"
	if os.Getenv("CHATAPP_HTTPS_PORT") != "" {
		httpsPort = os.Getenv("CHATAPP_HTTPS_PORT")
	}
	if os.Getenv("CHATAPP_HTTP_PORT") != "" {
		httpPort = os.Getenv("CHATAPP_HTTP_PORT")
	}
	if os.Getenv("CHATAPP_TLS_PRIVATE_KEY") == "" || os.Getenv("CHATAPP_TLS_CERTIFICATE") == "" {
		log.Fatal("Need to set CHATAPP_TLS_PRIVATE_KEY and CHATAPP_TLS_CERTIFICATE to continue")
	}
	certFile := os.Getenv("CHATAPP_TLS_CERTIFICATE")
	keyFile := os.Getenv("CHATAPP_TLS_PRIVATE_KEY")

	log.Println("hello")
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		log.Println("HTTPS")
		defer wg.Done()
		server := &http.Server{
			Addr:         ":" + httpsPort,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			TLSConfig:    tlsConfig,
			Handler:      chatServer.SetupRouter(),
		}
		log.Println("Starting HTTPS server on " + server.Addr)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		log.Println("HTTP")
		defer wg.Done()
		// Redirect all HTTP requests to the HTTPS server
		server := &http.Server{
			Addr:         ":" + httpPort,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Connection", "close")
				redirectUrl := &url.URL{}
				redirectUrl.Scheme = "https"
				redirectUrl.Host = r.Host
				if strings.Contains(r.Host, ":") {
					// If the user requested a custom HTTP port, reconstruct the host:port combination
					// using the given HTTPS address.
					redirectUrl.Host = strings.Split(r.Host, ":")[0] + ":" + httpsPort
				}
				log.Println(redirectUrl.String())
				http.Redirect(w, r, redirectUrl.String(), http.StatusMovedPermanently)
			}),
		}
		log.Println("Starting HTTP server on " + server.Addr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	wg.Wait()
}
