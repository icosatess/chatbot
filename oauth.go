package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

var secrets = GetSecrets()
var oauth2Config = &oauth2.Config{
	ClientID:     secrets.ClientID,
	ClientSecret: secrets.ClientSecret,
	Scopes: []string{
		"user:read:chat", "user:write:chat", "user:bot",
	},
	Endpoint:    twitch.Endpoint,
	RedirectURL: "http://localhost:8082",
}

var errNoTokenReceived = errors.New("failed to get OAuth 2 token; check logs")

type callbackHandler struct {
	completion chan *oauth2.Token
}

func (h callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("OAuth callback handler invoked")
	code := r.FormValue("code")

	tok, tokErr := oauth2Config.Exchange(context.TODO(), code)
	if tokErr != nil {
		panic(tokErr)
	}

	h.completion <- tok
}

// DoOauthDance performs interactive OAuth2 authorization and returns an HTTP
// client that performs authed requests against the Twitch API.
func DoOauthDance(ctx context.Context) (*http.Client, error) {
	url := oauth2Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	if _, err := fmt.Printf("Go to auth URL: %s", url); err != nil {
		log.Fatalf("failed to prompt user to go to auth URL: %v", err)
	}

	channel := make(chan *oauth2.Token)
	h := callbackHandler{channel}
	callbackServer := &http.Server{
		Addr:    "127.0.0.1:8082",
		Handler: h,
	}
	go func() {
		if err := callbackServer.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
			// Normal shutdown occurred.
		} else if err != nil {
			log.Printf("got unexpected error listening on callback server, ignoring: %v", err)
		}
	}()
	t, ok := <-channel
	if err := callbackServer.Shutdown(ctx); err != nil {
		log.Printf("failed to shut down callback server gracefully, ignoring: %v", err)
	}
	if !ok {
		return nil, errNoTokenReceived
	}

	return oauth2Config.Client(ctx, t), nil
}
