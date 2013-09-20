package local

import (
	"fmt"
	"github.com/deet/picturelife-experimental-uploader/api"
	"log"
	"os"
)

func (appState *State) UpdateToken() {
	token := appState.Api.AccessToken.Token
	envToken := os.Getenv("PLTOKEN")
	if envToken != "" {
		log.Println("Using token from PLTOKEN environment variable.")
		token = envToken
	}

	if token == "" || appState.Api.CheckToken(token) == false {
		log.Println("No valid access token found. Please login. Existing token:", token)
		email, password := "", ""
		for email == "" || password == "" {
			fmt.Println("Email: ")
			fmt.Scanln(&email)
			fmt.Println("Password: ")
			fmt.Scanln(&password)
			appState.Api.AccessToken, _ = appState.Api.Login(email, password)
			if appState.Api.AccessToken.Token != "" {
				fmt.Printf("\n\nLogin sucessessful. \n\n", appState.Api.AccessToken.Token)
				break
			}
			os.Setenv("PLTOKEN", appState.Api.AccessToken.Token)
		}
		if appState.Api.AccessToken.Token == "" {
			panic("No access token")
		}
	} else {
		appState.Api.AccessToken = api.AccessToken{Token: token}
	}
	return
}
