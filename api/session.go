package api

import (
	"log"
	"net/url"
)

type AccessToken struct {
	Token        string
	UserId       string
	RefreshToken string
	Expires      float64
	Email        string
}

type LoginResponse struct {
	ApiResponse
	AccessToken
}

type TokenStatusResponse struct {
	Expired    bool
	ExpireTime int
	TokenType  string
}

type CheckTokenResponse struct {
	ApiResponse
	TokenStatus TokenStatusResponse
}

func (api *API) Login(email, password string) (token AccessToken, err error) {
	path := "oauth/access_token"

	api.AccessToken.Token = ""

	params := url.Values{}
	params.Add("email", email)
	params.Add("password", password)
	params.Add("grant_type", "password")
	params.Add("client_id", api.ClientId)
	params.Add("client_secret", api.ClientSecret)
	params.Add("client_uuid", "Open source install")

	log.Println("Params:", params)

	defer func() {
		if r := recover(); r != nil {
			if r == "Call failed" {
				log.Println("Login failed because call failed")
				return
			}
		}
	}()

	var loginResponse LoginResponse
	api.CallAndParseIntoWithOutput(path, params, &loginResponse, true)

	if loginResponse.Token == "" {
		panic("Login error")
	}
	token = AccessToken{
		Token:        loginResponse.Token,
		UserId:       loginResponse.UserId,
		RefreshToken: loginResponse.RefreshToken,
		Expires:      loginResponse.Expires,
		Email:        loginResponse.Email,
	}

	return
}

func (api *API) CheckToken(tokenString string) (validToken bool) {
	//return true
	path := "oauth/check_token"

	params := url.Values{}
	params.Add("access_token", tokenString)

	defer func() {
		if r := recover(); r != nil {
			if r == "Call failed" {
				log.Println("Check token failed because call failed")
				validToken = false
				return
			}
		}
	}()

	var checkTokenResponse CheckTokenResponse
	api.CallAndParseIntoWithOutput(path, params, &checkTokenResponse, false)

	if checkTokenResponse.TokenStatus.Expired == false {
		log.Println("Good token, returning true")
		validToken = true
		return true
	}

	log.Println("Bad token")
	validToken = false
	return
}
