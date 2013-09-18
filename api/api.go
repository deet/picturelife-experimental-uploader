package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type ClientCredentials struct {
	ClientId     string
	ClientSecret string
}

type API struct {
	AccessToken  AccessToken
	Host         string
	ServicesHost string
	Port         string
	ServicesPort string
	ClientCredentials
}

type APIInterface interface {
	MakeFullPath(path string) string
	PostWithToken(path string, values url.Values) (resp *http.Response, err error)
}

func (api *API) MakeFullPath(path string) string {
	port := ""
	if api.Port != "" {
		port = fmt.Sprintf(":%s", api.Port)
	}
	return fmt.Sprintf("%s%s/%s", api.Host, port, path)
}

func (api *API) PostWithToken(path string, values url.Values) (resp *http.Response, err error) {
	if api.AccessToken.Token != "" {
		values.Add("access_token", api.AccessToken.Token)
	}
	dest := api.MakeFullPath(path)
	//fmt.Println(dest)
	return http.PostForm(dest, values)
}

func (api *API) CallAndParseIntoWithOutput(path string, values url.Values, parseInto BasicResponse, output bool) int64 {
	startTime := time.Now()
	response, err := api.PostWithToken(path, values)
	endTime := time.Now()
	time := endTime.Sub(startTime).Nanoseconds() / 1000000
	//log.Println("Call time: ", time)
	if err != nil {
		fmt.Println(err)
		panic(err)
		return time
	}

	jsonOutputter := json.NewEncoder(os.Stdout)

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if output {
		fmt.Println(buf)
	}

	err = json.Unmarshal(buf.Bytes(), &parseInto)
	if err != nil {
		//fmt.Println(path, err)
		//fmt.Println(path, "Could not parse reponse: ", string(buf.Bytes()))
		return time
	}

	if output {
		jsonOutputter.Encode(parseInto)
	}

	if parseInto.GetStatus() != 20000 && parseInto.GetStatus() != 200 {
		panic("Call failed")
	}

	return time
}

func (api *API) CallAndParseInto(path string, values url.Values, parseInto BasicResponse) int64 {
	return api.CallAndParseIntoWithOutput(path, values, parseInto, false)
}

func (api *API) LoadClientCredentials(path string) error {
	log.Println("Loading client credentials from:", path)

	file, loadErr := ioutil.ReadFile(path)
	if loadErr != nil {
		log.Println("Could not load client credentials file.")
		return errors.New("Could not load credentials file.")
	}

	loadedCredentials := ClientCredentials{}
	jsonErr := json.Unmarshal(file, &loadedCredentials)
	if jsonErr != nil {
		log.Println("Could not parse client credentials file.")
		return errors.New("Could not load credentials file.")
	}

	api.ClientCredentials = loadedCredentials

	if api.ClientCredentials.ClientId == "" || api.ClientCredentials.ClientSecret == "" {
		log.Println("Client credentials blank.")
		return errors.New("Client credentials appear invalid.")
	}
	return nil
}
