package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

var errCouldNotSubscribe = errors.New("couldn't subscribe for listening to chat")
var errUnexpectedProtocolInteraction = errors.New("the Twitch API server engaged in an unexpected interaction")

func GetListeningConnection(client *http.Client) (*websocket.Conn, error) {
	ctx := context.TODO()
	c, _, cerr := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	if cerr != nil {
		log.Printf("error dialing for websocket, will bail: %v", cerr)
		return nil, errCouldNotSubscribe
	}

	messageType, createRequestBs, messageErr := c.Read(ctx)
	if messageErr != nil {
		log.Printf("error reading welcome message from websocket, will bail: %v", cerr)
		return nil, errCouldNotSubscribe
	}
	if messageType != websocket.MessageText {
		log.Printf("wanted message of type MessageText, but was %d, will bail", messageType)
		return nil, errUnexpectedProtocolInteraction
	}

	var wm WelcomeMessage
	var syntaxError *json.SyntaxError
	if err := json.Unmarshal(createRequestBs, &wm); errors.As(err, &syntaxError) {
		log.Printf("Twitch returned welcome message with syntax error: %v", syntaxError)
		return nil, errUnexpectedProtocolInteraction
	} else if err != nil {
		log.Printf("Unexpected error unmarshalling welcome message: %v", syntaxError)
		return nil, errCouldNotSubscribe
	}

	if wmt := wm.Metadata.MessageType; wmt != "session_welcome" {
		log.Printf("wanted message of type session_welcome, but was %s", wmt)
		return nil, errUnexpectedProtocolInteraction
	}

	createRequest := CreateSubscriptionRequestBody{
		Type:    "channel.chat.message",
		Version: "1",
		Condition: map[string]any{
			"broadcaster_user_id": "820137268",  // icosatess
			"user_id":             "1310854767", // icosabot
		},
		Transport: createSubscriptionTransport{
			Method:    "websocket",
			SessionID: wm.Payload.Session.ID,
		},
	}
	createRequestBs, createRequestBsErr := json.Marshal(createRequest)
	if createRequestBsErr != nil {
		log.Printf("failed to marshal create subscription request: %v", createRequestBsErr)
		return nil, errCouldNotSubscribe
	}
	buf := bytes.NewBuffer(createRequestBs)
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.twitch.tv/helix/eventsub/subscriptions", buf)
	if reqErr != nil {
		log.Printf("failed to create subscriptions request: %v", reqErr)
		return nil, errCouldNotSubscribe
	}

	secrets := GetSecrets()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", secrets.ClientID)
	resp, respErr := client.Do(req)
	if respErr != nil {
		log.Printf("failed to post to subscriptions endpoint: %v", respErr)
		return nil, errCouldNotSubscribe
	}
	if resp.StatusCode != http.StatusAccepted {
		body, bodyErr := io.ReadAll(resp.Body)
		if bodyErr != nil {
			// If body can't be read, just put the error message in its place.
			body = fmt.Appendf(nil, "error reading body: %v", bodyErr)
		}
		log.Printf("Wanted HTTP status 202, but was %s: %s", resp.Status, body)
		return nil, errUnexpectedProtocolInteraction
	}

	return c, nil
}

func SubscribeForUpdates(client *http.Client, conn *websocket.Conn) {
	ctx := context.TODO()

	for {
		messageType, messageBytes, messageErr := conn.Read(ctx)
		if messageErr != nil {
			// This probably includes intentional closing from Twitch
			log.Printf("failed to read message from Twitch, ignoring: %v", messageErr)
			continue
		}
		if messageType != websocket.MessageText {
			log.Printf("wanted message type MessageText, but was %d, ignoring", messageType)
			continue
		}

		var nm NotificationMessage
		if err := json.Unmarshal(messageBytes, &nm); err != nil {
			log.Printf("unexpected error unmarshalling message from Twitch, ignoring: %v", err)
			continue
		}
		if nm.Metadata.MessageType == "session_keepalive" {
			// Cool story bro.
			// TODO: update timeLastMessageReceived and add client disconnection logic
			continue
		} else if nm.Metadata.MessageType != "notification" {
			log.Printf("expecting notification message, but got %s, ignoring", string(messageBytes))
			continue
		}

		log.Printf("Message from %s: %s (message ID %s)", nm.Payload.Event.ChatterUserName, nm.Payload.Event.Message.Text, nm.Payload.Event.MessageID)

		trimmed := strings.TrimSpace(nm.Payload.Event.Message.Text)
		if strings.EqualFold(trimmed, "!source") {
			SendCurrentEditorURL(client)
		}
	}
}
