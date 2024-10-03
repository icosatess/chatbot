package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2/clientcredentials"
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
	cfg := clientcredentials.Config{
		ClientID:     secrets.ClientID,
		ClientSecret: secrets.ClientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
		Scopes: []string{
			"user:write:chat",
			"user:bot",
			"channel:bot",
		},
	}

	client := cfg.Client(context.Background())

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
