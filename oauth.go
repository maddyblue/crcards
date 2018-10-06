package main

import (
	"bytes"
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

func NewOAuthWrapper(
	config *oauth2.Config, handler http.Handler, callbackPath, emailDomain string,
) http.Handler {
	sc := newSecureCookie(config)
	rand := NewPseudoRand()
	var lock sync.Mutex

	const cookieName = "d"
	setCookie := func(w http.ResponseWriter, r *http.Request, value map[string]interface{}) error {
		encoded, err := sc.Encode(cookieName, value)
		if err != nil {
			return err
		}
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    encoded,
			Path:     "/",
			Secure:   r.URL.Scheme == "https",
			HttpOnly: true,
			MaxAge:   60 * 60 * 24 * 30, // 30 days
		})
		return nil
	}
	getCookie := func(r *http.Request) (map[string]interface{}, error) {
		cookie, err := r.Cookie(cookieName)
		m := map[string]interface{}{}
		if err == http.ErrNoCookie {
			return m, nil
		} else if err != nil {
			return nil, err
		}
		err = sc.Decode(cookieName, cookie.Value, &m)
		return m, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		cookie, err := getCookie(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		} else if cookie["state"] != r.FormValue("state") {
			http.Error(w, "bad state", 400)
			return
		}
		tok, err := config.Exchange(oauth2.NoContext, r.FormValue("code"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		id := tok.Extra("id_token")
		switch id := id.(type) {
		case string:
			email, err := emailFromIdToken(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !strings.HasSuffix(email, fmt.Sprintf("@%s", emailDomain)) {
				http.Error(w, fmt.Sprintf("unknown email domain: %s", email), 400)
				return
			}
			delete(cookie, "state")
			cookie["email"] = email
			cookie["expire"] = tok.Expiry
			cookie["token"] = tok
			redir, ok := cookie["redirect"].(string)
			if !ok {
				redir = "/"
			}
			delete(cookie, "redirect")
			if err := setCookie(w, r, cookie); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			http.Redirect(w, r, redir, 302)
		default:
			http.Error(w, "no id token", 500)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := getCookie(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if expire, ok := cookie["expire"].(time.Time); ok {
			if expire.After(time.Now()) {
				handler.ServeHTTP(w, r)
				return
			}
		}

		lock.Lock()
		state := strconv.Itoa(rand.Int())
		lock.Unlock()
		cookie["state"] = state
		cookie["redirect"] = r.URL.String()
		if err := setCookie(w, r, cookie); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		url := config.AuthCodeURL(state)
		http.Redirect(w, r, url, 307)
	})
	return mux
}

// NewPseudoSeed generates a seed from crypto/rand.
func NewPseudoSeed() int64 {
	var seed int64
	err := binary.Read(crypto_rand.Reader, binary.LittleEndian, &seed)
	if err != nil {
		panic(fmt.Sprintf("could not read from crypto/rand: %s", err))
	}
	return seed
}

// NewPseudoRand returns an instance of math/rand.Rand seeded from crypto/rand
// and its seed so we can easily and cheaply generate unique streams of
// numbers. The created object is not safe for concurrent access.
func NewPseudoRand() *rand.Rand {
	seed := NewPseudoSeed()
	return rand.New(rand.NewSource(seed))
}

func newSecureCookie(config *oauth2.Config) *securecookie.SecureCookie {
	var b bytes.Buffer
	b.WriteString(config.ClientID)
	b.WriteString(config.ClientSecret)
	sha := sha256.Sum256(b.Bytes())
	return securecookie.New(sha[:16], sha[16:32])
}

// From https://github.com/bitly/oauth2_proxy/blob/master/providers/google.go
func emailFromIdToken(idToken string) (string, error) {
	// id_token is a base64 encode ID token payload
	// https://developers.google.com/accounts/docs/OAuth2Login#obtainuserinfo
	jwt := strings.Split(idToken, ".")
	jwtData := strings.TrimSuffix(jwt[1], "=")
	b, err := base64.RawURLEncoding.DecodeString(jwtData)
	if err != nil {
		return "", err
	}

	var email struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	err = json.Unmarshal(b, &email)
	if err != nil {
		return "", err
	}
	if email.Email == "" {
		return "", errors.New("missing email")
	}
	if !email.EmailVerified {
		return "", fmt.Errorf("email %s not listed as verified", email.Email)
	}
	return email.Email, nil
}
