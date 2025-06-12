package main

import (
	"context"
	"errors"
	"log"

	"github.com/coder/websocket"
)

func main() {
	ctx := context.TODO()

	client, clientErr := DoOauthDance(ctx)
	if errors.Is(clientErr, errNoTokenReceived) {
		log.Fatalf("failed to get token, check logs")
	} else if clientErr != nil {
		log.Fatalf("unexpected error getting OAuth2 tokens: %v", clientErr)
	}

	conn, connErr := GetListeningConnection(client)
	if errors.Is(connErr, errCouldNotSubscribe) {
		log.Printf("failed to subscribe for listening to chat: %v", connErr)
	} else if errors.Is(connErr, errUnexpectedProtocolInteraction) {
		log.Panicf("failed to subscribe for listening to chat due to unexpected interaction with Twitch: %v", connErr)
	} else if connErr != nil {
		log.Panicf("failed to subscribe to chat due to unexpected error: %v", connErr)
	}
	defer conn.Close(websocket.StatusNormalClosure, "Bye")

	d := dispatcher{client}
	SubscribeForUpdates(ctx, conn, d)
}
