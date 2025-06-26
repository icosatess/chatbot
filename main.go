package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/coder/websocket"
)

var broadcasterUsername = os.Getenv("BROADCASTER_USERNAME")
var botUsername = os.Getenv("BOT_USERNAME")
var minimapServerAddress = os.Getenv("MINIMAP_SERVER_ADDRESS")
var codeServerAddress = os.Getenv("CODE_SERVER_ADDRESS")

func main() {
	ctx := context.TODO()

	client, clientErr := MakeAuthorizedClient(ctx)
	if errors.Is(clientErr, errNoTokenReceived) {
		log.Fatalf("failed to get token, check logs")
	} else if clientErr != nil {
		log.Fatalf("unexpected error getting OAuth2 tokens: %v", clientErr)
	}

	conn, connErr := GetListeningConnection(ctx, client)
	if errors.Is(connErr, errCouldNotSubscribe) {
		log.Printf("failed to subscribe for listening to chat: %v", connErr)
	} else if errors.Is(connErr, errUnexpectedProtocolInteraction) {
		log.Fatalf("failed to subscribe for listening to chat due to unexpected interaction with Twitch: %v", connErr)
	} else if connErr != nil {
		log.Fatalf("failed to subscribe to chat due to unexpected error: %v", connErr)
	}
	defer conn.Close(websocket.StatusNormalClosure, "Bye")

	d := dispatcher{client}
	SubscribeForUpdates(ctx, conn, d)
}
