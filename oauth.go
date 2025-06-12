package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

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
	if r.URL.Path != "/" {
		http.Error(w, "This server supports callbacks only", http.StatusNotFound)
		return
	}
	log.Printf("OAuth callback handler invoked")
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	tok, tokErr := oauth2Config.Exchange(context.TODO(), code)
	if tokErr != nil {
		panic(tokErr)
	}

	h.completion <- tok
}

// MakeAuthorizedClient performs interactive OAuth2 authorization and returns an
// HTTP client that performs authed requests against the Twitch API.
func MakeAuthorizedClient(ctx context.Context) (*http.Client, error) {
	// Check for cached token on disk
	token, tokenErr := GetTokenFromDisk(ctx)
	if errors.Is(tokenErr, errNoCachedToken) {
		// A dance is necessary
	} else if tokenErr != nil {
		panic(tokenErr)
	} else {
		// TODO: Check whether the tokens are still valid
		return oauth2Config.Client(ctx, token), nil
	}

	// If absent, do a dance
	token, tokenErr = GetTokenByDancing(ctx)
	if tokenErr != nil {
		panic(tokenErr)
	}

	// Save the token to disk
	if err := SaveTokenToDisk(ctx, token); err != nil {
		panic(err)
	}

	// TODO: Check whether the tokens are still valid
	return oauth2Config.Client(ctx, token), nil
}

var errNoCachedToken = errors.New("no cached token")

func GetTokenFromDisk(ctx context.Context) (*oauth2.Token, error) {
	bs, bserr := os.ReadFile("tokens.json")
	if errors.Is(bserr, os.ErrNotExist) {
		return nil, errNoCachedToken
	}
	if bserr != nil {
		panic(bserr)
	}
	var token = &oauth2.Token{}
	if err := json.Unmarshal(bs, token); err != nil {
		panic(err)
	}
	return token, nil
}

func SaveTokenToDisk(ctx context.Context, token *oauth2.Token) error {
	bs, bserr := json.Marshal(token)
	if bserr != nil {
		panic(bserr)
	}
	if err := os.WriteFile("tokens.json", bs, 0o777); err != nil {
		panic(err)
	}
	return nil
}

func GetTokenByDancing(ctx context.Context) (*oauth2.Token, error) {
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

	return t, nil
}
