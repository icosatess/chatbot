package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type getUsersResponseBody struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

// GetBotsUserID returns the user ID of the broadcaster and the user ID of the bot.
func GetBotsUserID(client *http.Client, clientID string) (string, string) {
	// TODO: Make this safer
	req, reqErr := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s&login=%s", broadcasterUsername, botUsername), nil)
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
		case strings.EqualFold(entry.Login, broadcasterUsername):
			broadcasterID = entry.ID
		case strings.EqualFold(entry.Login, botUsername):
			botID = entry.ID
		}
	}

	// TODO: actually verify both were set
	return broadcasterID, botID
}
