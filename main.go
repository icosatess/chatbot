package main

import (
	"context"
	"errors"
	"log"
)

func main() {
	client, clientErr := DoOauthDance(context.TODO())
	if errors.Is(clientErr, errNoTokenReceived) {
		log.Fatalf("failed to get token, check logs")
	} else if clientErr != nil {
		log.Fatalf("unexpected error getting OAuth2 tokens: %v", clientErr)
	}

	SubscribeForUpdates(client)
}
