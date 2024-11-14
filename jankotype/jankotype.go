package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ComponentData struct {
	Component    string `json:"component"`
	RelativePath string `json:"relativePath"`
}

const baseURL = "https://icosatess.tail12901.ts.net"

var componentNameToWorkspaceFolderName = map[string]string{
	"ui":         "minimapui",
	"server":     "minimapsrv",
	"extension":  "minimapext",
	"codeserver": "codesrv",
	"chatbot":    "chatbot",
}

func main() {
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
		if _, err := fmt.Printf("No file is open\n"); err != nil {
			panic(err)
		}
	} else if componentData.Component == "" && componentData.RelativePath != "" {
		// Open file is outside any workspace folder
		if _, err := fmt.Printf("Current source file isn't available on the code viewer\n"); err != nil {
			panic(err)
		}
	} else if componentData.Component != "" && componentData.RelativePath == "" {
		// Open file does not have a file path
		if _, err := fmt.Printf("Current source file isn't available on the code viewer\n"); err != nil {
			panic(err)
		}
	} else {
		workspaceFolderName := componentNameToWorkspaceFolderName[componentData.Component]
		if _, err := fmt.Printf("%s/%s/%s\n", baseURL, workspaceFolderName, componentData.RelativePath); err != nil {
			panic(err)
		}
	}
}
