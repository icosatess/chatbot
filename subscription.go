package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

var timeLastMessageReceived time.Time

type WelcomeMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "session_welcome"
	} `json:"metadata"`
	Payload struct {
		Session struct {
			ID                      string `json:"id"`
			KeepaliveTimeoutSeconds int    `json:"keepalive_timeout_seconds"`
		} `json:"session"`
	} `json:"payload"`
}

type KeepaliveMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "session_keepalive"
	} `json:"metadata"`
	Payload struct{} `json:"payload"`
}

type NotificationMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "notification"
	} `json:"metadata"`
	Payload struct {
		Event ChannelChatMessageEvent `json:"event"`
	} `json:"payload"`
}

type ChannelChatMessageEvent struct {
	ChatterUserName string `json:"chatter_user_name"`
	MessageID       string `json:"message_id"`
	Message         struct {
		Text string `json:"text"`
	} `json:"message"`
}

/*
{
  "metadata": {
    "message_id": "96a3f3b5-5dec-4eed-908e-e11ee657416c",
    "message_type": "session_welcome",
    "message_timestamp": "2023-07-19T14:56:51.634234626Z"
  },
  "payload": {
    "session": {
      "id": "AQoQILE98gtqShGmLD7AM6yJThAB",
      "status": "connected",
      "connected_at": "2023-07-19T14:56:51.616329898Z",
      "keepalive_timeout_seconds": 10,
      "reconnect_url": null
    }
  }
}
*/

// Create subscription POST https://api.twitch.tv/helix/eventsub/subscriptions

type CreateSubscriptionTransport struct {
	Method    string `json:"method"` // websocket
	SessionID string `json:"session_id"`
}

type CreateSubscriptionRequestBody struct {
	Type      string                      `json:"type"`      // "channel.chat.message"
	Version   string                      `json:"version"`   // 1
	Condition map[string]any              `json:"condition"` // broadcaster_user_id: {icosatess}, user_id: {icosabot}
	Transport CreateSubscriptionTransport `json:"transport"`
}

func SubscribeForUpdates() {
	c, _, cerr := websocket.Dial(context.TODO(), "wss://eventsub.wss.twitch.tv/ws", nil)
	if cerr != nil {
		panic(cerr)
	}
	defer c.Close(websocket.StatusNormalClosure, "Bye")

	t, bs, rerr := c.Read(context.TODO())
	if rerr != nil {
		panic(rerr)
	}
	if t != websocket.MessageText {
		log.Panicf("wanted message of type MessageText, but was %d", t)
	}
	timeLastMessageReceived = time.Now()

	var wm WelcomeMessage
	if err := json.Unmarshal(bs, &wm); err != nil {
		panic(err)
	}

	if wmt := wm.Metadata.MessageType; wmt != "session_welcome" {
		log.Panicf("wanted message of type session_welcome, but was %s", wmt)
	}

	// TODO: get a user access token for... icosabot?
	// Scopes: user:read:chat
	createRequest := CreateSubscriptionRequestBody{
		Type:    "channel.chat.message",
		Version: "1",
		Condition: map[string]any{
			"broadcaster_user_id": "820137268",  // icosatess
			"user_id":             "1310854767", // icosabot
		},
		Transport: CreateSubscriptionTransport{
			Method:    "webhook",
			SessionID: wm.Payload.Session.ID,
		},
	}
	bs, bserr := json.Marshal(createRequest)
	if bserr != nil {
		panic(bserr)
	}
	buf := bytes.NewBuffer(bs)
	resp, respErr := http.Post("https://api.twitch.tv/helix/eventsub/subscriptions", "application/json", buf)
	if respErr != nil {
		panic(respErr)
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Panicf("Wanted HTTP status 202, but was %d", resp.StatusCode)
	}

	for {
		t, bs, err := c.Read(context.TODO())
		if err != nil {
			// This probably includes intentional closing from Twitch
			panic(err)
		}
		if t != websocket.MessageText {
			log.Panicf("wanted message type MessageText, but was %d", t)
		}

		var nm NotificationMessage
		if err := json.Unmarshal(bs, &nm); err != nil {
			panic(err)
		}
		if nm.Metadata.MessageType != "notification" {
			log.Panicf("expecting notification message, but got %s", string(bs))
		}
		timeLastMessageReceived = time.Now()

		log.Printf("Message from %s: %s (message ID %s)", nm.Payload.Event.ChatterUserName, nm.Payload.Event.Message.Text, nm.Payload.Event.MessageID)

		trimmed := strings.TrimSpace(nm.Payload.Event.Message.Text)
		if strings.EqualFold(trimmed, "!source") {
			// TODO: reply to the message with a link to the source code
		}
	}
}
