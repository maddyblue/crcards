package main

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//go:generate yarn --cwd frontend build:css
//go:generate yarn --cwd frontend purge:css
//go:generate yarn --cwd frontend build
//go:generate esc -o static.go -prefix frontend/build -ignore \.map frontend/build

type Specification struct {
	// HTTP listen address if Autocert not specified.
	Addr string

	// Let's Encrypt domain name to use. Enables HTTPS on :443 and HTTP on :80. Ignores Addr.
	Autocert string
	// DirCache connection string for autocert db cache.
	DirCache string

	// BambooHR API key and domain.
	APIKey       string
	BambooDomain string

	// OAuth client id, secret, and redirect_url address (redirect should exclude the path: 'http://localhost:4001').
	OAuthClientID     string
	OAuthClientSecret string
	Redirect          string
	// Email domain to restrict oauth logins to.
	EmailDomain string
}

func init() {
	gob.Register(time.Time{})
	gob.Register(oauth2.Token{})
}

func main() {
	var spec Specification
	err := envconfig.Process("CARDS", &spec)
	if err != nil {
		log.Fatal(err)
	}
	const callbackPath = "/oauth/callback"
	if spec.Redirect == "" {
		spec.Redirect = fmt.Sprintf("http://%s", spec.Addr)
	}

	bambooClient := NewBambooClient(spec.BambooDomain, spec.APIKey)
	var dirCache []byte
	var dirTime time.Time
	var lock sync.Mutex

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(FS(false)))
	mux.HandleFunc("/api/get-employees", func(w http.ResponseWriter, r *http.Request) {
		if spec.APIKey == "" {
			b, _ := ioutil.ReadFile("f")
			w.Write(b)
			return
		}

		lock.Lock()
		defer lock.Unlock()

		if dirTime.Before(time.Now()) {
			dir, err := bambooClient.EmployeeDirectory(r.Context())
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			var b bytes.Buffer
			if err := json.NewEncoder(&b).Encode(dir.Employees); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			dirCache = b.Bytes()
			dirTime = time.Now().Add(time.Hour)
		}

		w.Write(dirCache)
	})
	var handler http.Handler = mux
	if spec.OAuthClientID != "" {
		oauth := &oauth2.Config{
			ClientID:     spec.OAuthClientID,
			ClientSecret: spec.OAuthClientSecret,
			RedirectURL:  spec.Redirect + callbackPath,
			Scopes:       []string{"email"},
			Endpoint:     google.Endpoint,
		}
		handler = NewOAuthWrapper(oauth, mux, callbackPath, spec.EmailDomain)
	}

	if spec.Autocert != "" {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(spec.Autocert),
			Cache:      autocert.DirCache(spec.DirCache),
		}
		tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}
		go func() {
			fmt.Println("listening on :http for redirect")
			log.Fatal(http.ListenAndServe(":http", m.HTTPHandler(nil)))
		}()
		s := &http.Server{
			Addr:      ":https",
			TLSConfig: tlsConfig,
			Handler:   handler,
		}
		fmt.Println("listening on :https for", spec.Autocert)
		log.Fatal(s.ListenAndServeTLS("", ""))
	} else {
		fmt.Printf("listening on http://%s\n", spec.Addr)
		log.Fatal(http.ListenAndServe(spec.Addr, handler))
	}
}
