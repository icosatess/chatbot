package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
)

type ComponentData struct {
	Component    string `json:"component"`
	RelativePath string `json:"relativePath"`
}

func MakeCodeBrowserURL() (*url.URL, error) {
	// TODO: Make this safer
	resp, respErr := http.Get(minimapServerAddress + "/component/")
	if respErr != nil {
		panic(respErr)
	}
	bs, bsErr := io.ReadAll(resp.Body)
	if bsErr != nil {
		panic(bsErr)
	}
	var componentData ComponentData
	if err := json.Unmarshal(bs, &componentData); err != nil {
		panic(err)
	}
	if componentData.Component == "" && componentData.RelativePath == "" {
		// No data available
		return nil, errors.New("no file is open")
	} else if componentData.Component == "" && componentData.RelativePath != "" {
		// Open file is outside any workspace folder
		return nil, errors.New("current source file isn't available on the code viewer")
	} else if componentData.Component != "" && componentData.RelativePath == "" {
		// Open file does not have a file path
		return nil, errors.New("current source file isn't available on the code viewer")
	} else {
		workspaceFolderName := componentData.Component
		p, perr := url.JoinPath(workspaceFolderName, componentData.RelativePath)
		if perr != nil {
			panic(perr)
		}
		// TODO: Make this safer
		u, uerr := url.Parse(codeServerAddress + "/" + p)
		if uerr != nil {
			panic(uerr)
		}
		return u, nil
	}
}

func SendCurrentEditorURL(client *http.Client) {
	secrets := GetSecrets()

	u, uerr := MakeCodeBrowserURL()
	if uerr != nil {
		log.Printf("couldn't figure out what URL to send to chat")
		return
	}

	broadcasterUserID, botUserID := GetBotsUserID(client, secrets.ClientID)

	messageBody := SendChatMessageRequestBody{
		BroadcasterID: broadcasterUserID,
		SenderID:      botUserID,
		Message:       u.String(),
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

	if resp.StatusCode != 200 {
		log.Printf("Wanted status code 200, but was %d", resp.StatusCode)
		bs, _ := io.ReadAll(resp.Body)
		log.Printf("Response body was %s", string(bs))
		panic(resp)
	}

	defer resp.Body.Close()
}
