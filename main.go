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

	"github.com/eshyong/chatapp/chat"
)

func main() {
	httpPort := os.Getenv("CHATAPP_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	httpsPort := os.Getenv("CHATAPP_HTTPS_PORT")
	if httpsPort == "" {
		httpsPort = "8443"
	}

	// Create a wait group to manage cleanup of both server routines
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		runAppServer(wg, httpsPort)
	}()
	go func() {
		runRedirectingServer(wg, httpPort, httpsPort)
	}()
	wg.Wait()
}

// Creates an HTTPS server which runs the main app
func runAppServer(wg *sync.WaitGroup, httpsPort string) {
	// Make sure the WaitGroup is signaled when the server object gets cleaned up
	defer wg.Done()

	// TLS setup
	tlsConfig := createDefaultTlsConfig()
	certFile := os.Getenv("CHATAPP_TLS_CERTIFICATE")
	keyFile := os.Getenv("CHATAPP_TLS_PRIVATE_KEY")

	if certFile == "" || keyFile == "" {
		log.Fatal("Need to set CHATAPP_TLS_PRIVATE_KEY and CHATAPP_TLS_CERTIFICATE to continue")
	}

	// Secret keys for cookies
	hashKey := os.Getenv("CHATAPP_COOKIE_SECRET_HASH_KEY")
	if hashKey == "" {
		log.Fatal("Need to set CHATAPP_COOKIE_SECRET_HASH_KEY")
	}
	if len(hashKey) != 32 && len(hashKey) != 64 {
		log.Println("Warning: CHATAPP_COOKIE_SECRET_HASH_KEY should be 32 or 64 bytes")
	}
	blockKey := os.Getenv("CHATAPP_COOKIE_SECRET_BLOCK_KEY")
	if blockKey == "" {
		log.Fatal("Need to set CHATAPP_COOKIE_SECRET_BLOCK_KEY")
	}
	if len(blockKey) != 16 && len(blockKey) != 24 && len(blockKey) != 32 {
		log.Println("Warning: CHATAPP_COOKIE_SECRET_BLOCK_KEY should be 16, 24, 32 bytes")
	}

	// App setup
	app := chat.NewApp(hashKey, blockKey)
	server := &http.Server{
		Addr:         ":" + httpsPort,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig,
		Handler:      app.SetupRouter(),
	}
	log.Println("Starting HTTPS server on " + server.Addr)
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatal(err)
	}
}

func createDefaultTlsConfig() *tls.Config {
	// TLS config taken from Filippo Valsorda's blog post:
	// https://blog.cloudflare.com/exposing-go-on-the-internet/
	return &tls.Config{
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
}

// Creates an HTTP server which redirects all requests to the main HTTPS server
func runRedirectingServer(wg *sync.WaitGroup, httpPort, httpsPort string) {
	defer wg.Done()

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
				// Redirect all HTTP requests to the HTTPS server.
				// If the developer set a custom HTTP port, reconstruct the host:port combination
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
}
