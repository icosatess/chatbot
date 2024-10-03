package main

import (
	"encoding/json"
	"io"
	"os"
)

type Secrets struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func GetSecrets() Secrets {
	f, ferr := os.Open("secrets.json")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()

	secstr, secstrErr := io.ReadAll(f)
	if secstrErr != nil {
		panic(secstrErr)
	}

	var secrets Secrets
	if err := json.Unmarshal(secstr, &secrets); err != nil {
		panic(err)
	}

	return secrets
}
