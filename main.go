package main

import (
	"./local"
	"./web"
	"flag"
	"fmt"
	"github.com/cratonica/trayhost"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var hostFlag = flag.String("host", "http://localhost", "host to test")
var portFlag = flag.String("port", "3000", "port number on host")
var envFlag = flag.String("env", "production", "blank (specify host and port), production, or staging")
var clientfileFlag = flag.String("clientfile", "client.json", "Path to client credentials JSON file. Needs to be a JSON object with two string values: ClientId and ClientSecret")
var concurrentUploadsFlag = flag.Int("concurrent", 4, "maximum number of concurrent uploads")
var watchFlag = flag.Bool("watch", false, "watch a directory instead of uploading it immediately")
var configFlag = flag.Bool("config", false, "enable config mode")
var uploadRawFlag = flag.String("upload-raw", "", "upload RAW images?")
var uploadImagesFlag = flag.String("upload-images", "", "upload images?")
var uploadVideosFlag = flag.String("upload-videos", "", "upload videos?")
var guiFlag = flag.Bool("gui", true, "Enable GUI in CLI mode")

func init() {
	flag.Parse()
}

func processUploads(appState *local.State, mainWg *sync.WaitGroup) {
	defer mainWg.Done()

	// Setup concurrency limiting channel
	for i := 0; i < *concurrentUploadsFlag; i++ {
		appState.MaxUploadsChan <- 1
	}

	log.Println("Concurrent uploads:", *concurrentUploadsFlag)

	var uploadWg sync.WaitGroup

	watchHappening, directHappening := true, true
	for {
		if !watchHappening && !directHappening {
			log.Println("Waiting for upload routines to finish")
			uploadWg.Wait()
			return
		}
		select {
		case incomingFile, watchOk := <-appState.WatchFileChan:
			if watchOk {
				<-appState.MaxUploadsChan
				uploadWg.Add(1)
				go appState.HandleFile(incomingFile, &uploadWg)
			} else {
				//log.Println("Watch channel is closed")
				watchHappening = false
			}
		case incomingFile, directOk := <-appState.DirectFileChan:
			if directOk {
				<-appState.MaxUploadsChan
				uploadWg.Add(1)
				go appState.HandleFile(incomingFile, &uploadWg)
			} else {
				//log.Println("Direct channel closed")
				directHappening = false
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func configApiConnect(appState *local.State) {
	switch *envFlag {
	case "production":
		log.Println("USING PRODUCTION API")
		appState.Api.Host = "https://api.picturelife.com"
		appState.Api.ServicesHost = "https://services.picturelife.com"
	case "staging":
		appState.Api.Host = "https://api-staging.picturelife.com"
		appState.Api.ServicesHost = "https://services-staging.picturelife.com"
	case "development":
		appState.Api.Host = "http://localhost"
		appState.Api.ServicesHost = "http://localhost"
		appState.Api.Port = "3000"
		appState.Api.ServicesPort = "3001"
	default:
		panic("No environment set")
	}

}

func configState(appState *local.State) {
	if *uploadRawFlag != "" {
		if *uploadRawFlag == "true" {
			appState.UploadRaw = true
		} else if *uploadRawFlag == "false" {
			appState.UploadRaw = false
		}
	}
	if *uploadImagesFlag != "" {
		if *uploadImagesFlag == "true" {
			appState.UploadImages = true
		} else if *uploadImagesFlag == "false" {
			appState.UploadImages = false
		}
	}
	if *uploadVideosFlag != "" {
		if *uploadVideosFlag == "true" {
			appState.UploadVideo = true
		} else if *uploadVideosFlag == "false" {
			appState.UploadVideo = false
		}
	}
	appState.Save()
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	runtime.LockOSThread()

	go func() {

		var appState local.State
		filePath := flag.Arg(0)

		statePath := fmt.Sprintf("data/data_%s.json", *envFlag)
		appState = local.NewState(statePath)

		if *configFlag {
			configState(&appState)
			if filePath == "" {
				return
			}
		}

		appState.Load()
		configApiConnect(&appState)
		credentialsErr := appState.Api.LoadClientCredentials(*clientfileFlag)
		if credentialsErr != nil {
			panic("API CREDENTIALS ARE REQUIRED")
		}
		appState.UpdateToken()
		appState.Save()

		log.Println("Have token:", appState.Api.AccessToken.Token)

		appState.WatchFileChan = make(chan local.File, 1)
		appState.DirectFileChan = make(chan local.File, 1)
		appState.MaxUploadsChan = make(chan int, *concurrentUploadsFlag)

		var mainWg sync.WaitGroup

		mainWg.Add(1)
		go processUploads(&appState, &mainWg)

		if filePath == "" {
			fmt.Printf("\nGUI MODE\n\n")
			fmt.Println("Visit: http://localhost:7111/ in your web browser.")
			fmt.Printf("\n\n")

			// This should be started after launching the upload routines
			go web.StartWebUi(&appState)

			appState.UpdateDirectoryWatchers()
			appState.UploadWatchedDirectories()
		} else {
			fmt.Printf("\nCLI MODE\n\n")
			filePath, err := filepath.Abs(filePath)
			if err != nil {
				panic("Could not determine absolute file path.")
			}

			passedFile, err := os.Open(filePath)
			if err != nil {
				panic("Could not check path.")
			}

			if *guiFlag {
				fmt.Println("GUI enabled. Visit: http://localhost:7111/ in your web browser.")
				go web.StartWebUi(&appState)
			}

			if *watchFlag {
				appState.WatchFilesystem(filePath)
			} else {
				log.Println("Not watching")
				if fileInfo, _ := passedFile.Stat(); fileInfo.IsDir() {
					log.Println("Passed directory")
					mainWg.Add(1)
					go func() {
						log.Println("Trying to upload directory")
						appState.UploadDirectory(filePath, appState.DirectFileChan)
						mainWg.Done()
					}()
				} else {
					log.Println("Passed file")
					mainWg.Add(1)
					go func() {
						log.Println("Trying to upload file")
						_, _, err := appState.UploadFile(filePath, appState.DirectFileChan)
						if err != nil {
							log.Println(err)
						}
						mainWg.Done()
					}()
				}
			}
		}

		//appState.DoneChan = make(chan int)

		log.Println("Waiting for main routines to finish")
		mainWg.Wait()

	}()

	// Enter the host system's event loop
	trayhost.EnterLoop("Open Picturelife", iconData)

	// This is only reached once the user chooses the Exit menu item
	fmt.Println("Exiting")

}
