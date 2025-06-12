package main

import "net/http"

type dispatcher struct {
	client *http.Client
}

func (d dispatcher) SendSourceURL() {
	SendCurrentEditorURL(d.client)
}
