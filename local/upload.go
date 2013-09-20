package local

import (
	"errors"
	"github.com/deet/picturelife-experimental-uploader/util"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func (appState *State) HandleFile(file File, uploadWg *sync.WaitGroup) {
	defer uploadWg.Done()
	defer func() { appState.MaxUploadsChan <- 1 }()

	if file.Signature == "" {
		log.Println("got empty signature for", file.Path)
		return
	}
	existingFile, fileExists := appState.GetFile(file.Signature)
	force := false
	if fileExists {
		force = (existingFile.Status == "retrying")
		//log.Println("file exists", existingFile.Status)
		if existingFile.Status == "uploaded" {
			if existingFile.PendingMediaId == "" && existingFile.MediaId == "" {
				force = true
			} else {
				return
			}
		}
	}

	pendingMediaId, mediaId := "", ""
	existingDeleted := false
	var err error

	if force {
		pendingMediaId, mediaId, existingDeleted, err = appState.Api.UploadForce(file.Path, file.Signature, true)
	} else {
		pendingMediaId, mediaId, existingDeleted, err = appState.Api.Upload(file.Path, file.Signature)
	}
	if err == nil {
		if pendingMediaId == "" && mediaId == "" {
			file.Status = "errored"
			log.Println("Upload failed: no pending media and no media ID")
		} else {
			file.PendingMediaId = pendingMediaId
			file.MediaId = mediaId
			file.Status = "uploaded"
			if mediaId != "" {
				log.Printf("File (%s) previously uploaded and processed. Media ID: %s\n", file.Path, mediaId)
				if existingDeleted {
					file.Status = "uploaded-deleted"
					log.Printf("File (%s) previously deleted. Media ID: %s\n", file.Path, mediaId)
				}
			} else if pendingMediaId != "" {
				log.Printf("File (%s) uploaded and processing. Pending media ID: %s\n", file.Path, pendingMediaId)
			}
		}
	} else {
		file.Status = "errored"
		log.Println("Upload failed:", err)
	}
	appState.SetFile(file)
	appState.Save()
}

func (state *State) visitFile(path string, info os.FileInfo, err error, c chan File, retrying bool) (retErr error) {
	if info.IsDir() {
		log.Println("Directory, skipping")
		return
	}

	// Check for upload
	// Check for type
	extension := strings.ToUpper(filepath.Ext(path))
	contains := func(ary []string, value string) bool {
		for _, elem := range ary {
			if elem == value {
				return true
			}
		}
		return false
	}
	recognizedFormat := false
	if contains(state.ImageExtensions, extension) {
		recognizedFormat = true
		if !state.UploadImages {
			log.Println("Found image, but image uploads are not enabled.")
			return
		}
	}
	if contains(state.RawExtensions, extension) {
		recognizedFormat = true
		if !state.UploadRaw {
			log.Println("Found RAW image, but RAW image uploads are not enabled.")
			return
		}
	}
	if contains(state.VideoExtensions, extension) {
		recognizedFormat = true
		if !state.UploadVideo {
			log.Println("Found video, but video uploads are not enabled.")
			return
		}
	}
	signature := util.CalculateSignature(path)
	file := File{
		Signature:           signature,
		Path:                path,
		Extension:           extension,
		Name:                filepath.Base(path),
		MissingOnFilesystem: false,
	}
	_, exists := state.GetFile(signature)

	if !exists {
		file.Status = "pending"
		state.SetFile(file)
	} else if retrying {
		//log.Println("!!!!! visitFile: exists")
		file.Status = "retrying"
		state.SetFile(file)
	}
	if !recognizedFormat {
		log.Println("Unrecognized format")
		file.Status = "rejected_format"
	}
	c <- file

	return
}

func (state *State) UploadFile(path string, c chan File) (found, uploaded int64, err error) {
	return state.uploadFileOrRetry(path, c, false)
}

func (state *State) RetryFile(path string, c chan File) (found, uploaded int64, err error) {
	return state.uploadFileOrRetry(path, c, true)
}

func (state *State) uploadFileOrRetry(path string, c chan File, retry bool) (found, uploaded int64, err error) {
	dir, err := os.Open(path)
	if err != nil {
		err = errors.New("Could not read path.")
		return
	}
	fileInfo, _ := dir.Stat()
	if fileInfo.IsDir() {
		err = errors.New("Directory")
		return
	}

	state.visitFile(path, fileInfo, err, c, retry)
	return
}

func (state *State) UploadDirectory(path string, c chan File) (found, uploaded int64, err error) {
	dir, err := os.Open(path)
	if err != nil {
		err = errors.New("Could not read path.")
		return
	}
	if fileInfo, _ := dir.Stat(); !fileInfo.IsDir() {
		err = errors.New("Not a directory")
		return
	}

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		log.Println("Checking file", path)
		return state.visitFile(path, info, err, c, false)
	})

	//close(c)
	return
}
