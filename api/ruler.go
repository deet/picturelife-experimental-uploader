package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	//"path"
	"strconv"
)

type RulerResponse struct {
	Status    int64
	Location  string
	Error     string
	Signature string
}

func (api *API) rulerUpload(filePath, localSig string, restart bool) (location, signature string, err error) {
	niceLog := func(parts ...string) {
		log.Println("RULER (", filePath, ") ", strings.Join(parts, " "))
	}

	defer func() {
		if e := recover(); e != nil {
			niceLog("Error:", e.(string))
		}
	}()

	file, err := os.Open(filePath)
	if err != nil {
		panic("Could not open file")
	}
	defer file.Close()

	client := &http.Client{}
	//fileName := path.Base(filePath)
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	extension := strings.ToUpper(filepath.Ext(filePath))
	fakeFilename := fmt.Sprintf("%s%s", localSig, extension)

	//log.Println("using fakeFilename for ruler:", fakeFilename)

	params := url.Values{}
	params.Add("access_token", api.AccessToken.Token)
	params.Add("filename", fakeFilename)
	params.Add("signature", localSig)

	url := fmt.Sprintf("%s/ruler?%s", api.ServicesHost, params.Encode())

	var bytesCompleted int64 = 0
	if restart {
		log.Println("Restarting RULER upload")
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			panic(fmt.Sprintln("Could not form DELETE request", err))
		}

		deleteResp, err := client.Do(req)
		if err != nil {
			panic(fmt.Sprintln("Error restarting Ruler upload:", err))
		}
		defer deleteResp.Body.Close()
	} else {
		resp, err := client.Head(url)
		if err != nil {
			panic(fmt.Sprintln("Could not HEAD url", err))
		}

		if resp.Header.Get("X-Ruler-Error") != "" {
			panic(fmt.Sprintln("Ruler HEAD error:", resp.Header["X-Ruler-Error"]))
		}

		bytesCompleted, err = strconv.ParseInt(resp.Header.Get("X-Ruler-Size"), 10, 64)
		if err != nil {
			niceLog("Could not parse ruler size: ", resp.Header.Get("X-Ruler-Size"))
		} else {
			niceLog("Bytes completed: ", strconv.Itoa(int(bytesCompleted)))
		}
		file.Seek(bytesCompleted, 0)
	}

	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		panic(fmt.Sprintln("Could not form request", err))
	}
	req.ContentLength = (fileSize - bytesCompleted)
	if bytesCompleted > 0 {
		niceLog("Resuming failed uploaded from byte:", strconv.Itoa(int(bytesCompleted)), " (filesize: ", strconv.Itoa(int(fileSize)), ")")
		contentRangeValue := fmt.Sprintf("bytes %d-%d/%d", bytesCompleted, fileSize, fileSize)
		niceLog("Setting Content-Range", contentRangeValue)
		req.Header.Set("Content-Range", contentRangeValue)
	} else {
		niceLog("Not setting Content-Range because bytesCompleted: ", strconv.Itoa(int(bytesCompleted)))
	}

	//log.Printf("RULE REQUEST: %#v", req)

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			for k, v := range resp.Header {
				for _, v2 := range v {
					niceLog("Error header ", k, " : ", v2)
				}
			}
		}
		niceLog("Error message", err.Error())
		err = errors.New(fmt.Sprintf("Error during Ruler upload: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	fmt.Println(url)
	fmt.Println(resp.Body)

	var parsedResponse RulerResponse

	err = json.Unmarshal(buf.Bytes(), &parsedResponse)
	if err != nil {
		err = errors.New("Ruler returned invalid response.")
		return
	}

	if parsedResponse.Status == 519256 {
		niceLog("Ruler calculated different signature. Upload must be retried.")
		err = errors.New("Ruler calculated different signature. Upload must be retried.")
		return
	}

	if parsedResponse.Error != "" {
		niceLog("Error ", strconv.Itoa(int(parsedResponse.Status)), parsedResponse.Error)
		err = errors.New(fmt.Sprintf("Error (%d) %s", parsedResponse.Status, parsedResponse.Error))
		return
	}

	if parsedResponse.Location == "" {
		panic("Location is missing.")
	}

	//niceLog("Received location from RULER:", parsedResponse.Location)
	//niceLog("Received signature from RULER:", parsedResponse.Signature)

	signature = parsedResponse.Signature
	location = parsedResponse.Location

	return
}
