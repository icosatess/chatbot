package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

type sendChatMessageRequestBody struct {
	BroadcasterID        string `json:"broadcaster_id"`
	SenderID             string `json:"sender_id"`
	Message              string `json:"message"`
	ReplyParentMessageID string `json:"reply_parent_message_id"`
}

func main() {
	secrets := GetSecrets()
	conf := &oauth2.Config{
		ClientID:     secrets.ClientID,
		ClientSecret: secrets.ClientSecret,
		Scopes: []string{
			"user:read:chat", "user:write:chat",
		},
		Endpoint:    twitch.Endpoint,
		RedirectURL: "http://localhost:8082",
	}

	url := conf.AuthCodeURL("", oauth2.AccessTypeOffline)
	if _, err := fmt.Printf("Go to auth URL: %s", url); err != nil {
		panic(err)
	}

	done := make(chan any)
	var callbackServer *http.Server

	oauthCallbackHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth callback handler invoked")
		code := r.FormValue("code")

		go func(code string) {
			log.Printf("Entered newly spawned goroutine")
			// Will all of this run as the server is shutting down?
			tok, tokErr := conf.Exchange(context.TODO(), code)
			if tokErr != nil {
				panic(tokErr)
			}

			client := conf.Client(context.TODO(), tok)

			broadcasterID, botID := GetBotsUserID(client, secrets.ClientID)

			messageBody := sendChatMessageRequestBody{
				BroadcasterID: broadcasterID,
				SenderID:      botID,
				Message:       "Hello, world!",
			}

			jreq, jreqErr := json.Marshal(messageBody)
			if jreqErr != nil {
				panic(jreqErr)
			}

			req, reqErr := http.NewRequest(http.MethodPost, "https://api.twitch.tv/helix/chat/messages", bytes.NewBuffer(jreq))
			if reqErr != nil {
				panic(reqErr)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Client-Id", secrets.ClientID)
			resp, respErr := client.Do(req)
			if respErr != nil {
				panic(respErr)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.Printf("Wanted status code 200, but was %d", resp.StatusCode)
				bs, _ := io.ReadAll(resp.Body)
				log.Printf("Response body was %s", string(bs))
				panic(resp)
			}

			SubscribeForUpdates()
			done <- nil
		}(code)

		callbackServer.Shutdown(context.TODO())
	}

	callbackServer = &http.Server{
		Addr:    "127.0.0.1:8082",
		Handler: http.HandlerFunc(oauthCallbackHandler),
	}
	callbackServer.ListenAndServe()
	<-done
}

type getUsersResponseBody struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

// GetBotsUserID returns the user ID of the broadcaster and the user ID of the bot.
func GetBotsUserID(client *http.Client, clientID string) (string, string) {
	req, reqErr := http.NewRequest(http.MethodGet, "https://api.twitch.tv/helix/users?login=icosatess&login=icosabot", nil)
	if reqErr != nil {
		panic(reqErr)
	}
	req.Header.Set("Client-Id", clientID)
	resp, respErr := client.Do(req)
	if respErr != nil {
		panic(respErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("GetBotsUserID: Wanted status code 200, but was %d", resp.StatusCode)
		bs, _ := io.ReadAll(resp.Body)
		log.Printf("GetBotsUserID: Response body was %s", string(bs))
		panic(resp)
	}

	bodyStr, bodyStrErr := io.ReadAll(resp.Body)
	if bodyStrErr != nil {
		panic(bodyStrErr)
	}

	log.Printf("Got response body from Get Users: %s", string(bodyStr))

	var users getUsersResponseBody
	if err := json.Unmarshal(bodyStr, &users); err != nil {
		panic(err)
	}

	var broadcasterID, botID string
	for _, entry := range users.Data {
		switch {
		case strings.EqualFold(entry.Login, "icosatess"):
			broadcasterID = entry.ID
		case strings.EqualFold(entry.Login, "icosabot"):
			botID = entry.ID
		}
	}

	// TODO: actually verify both were set
	return broadcasterID, botID
}
