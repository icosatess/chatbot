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

var componentNameToWorkspaceFolderName = map[string]string{
	"ui":         "minimapui",
	"server":     "minimapsrv",
	"extension":  "minimapext",
	"codeserver": "codesrv",
	"chatbot":    "chatbot",
}

func MakeCodeBrowserURL() (*url.URL, error) {
	resp, respErr := http.Get("http://127.0.0.1:8081/component/")
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
		workspaceFolderName := componentNameToWorkspaceFolderName[componentData.Component]
		p, perr := url.JoinPath(workspaceFolderName, componentData.RelativePath)
		if perr != nil {
			panic(perr)
		}
		u := &url.URL{
			Scheme: "https",
			Host:   "icosatess.tail12901.ts.net",
			Path:   p,
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

	messageBody := SendChatMessageRequestBody{
		BroadcasterID: "820137268",  // icosatess
		SenderID:      "1310854767", // icosabot
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
