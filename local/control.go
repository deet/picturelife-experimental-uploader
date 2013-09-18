package local

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Request struct {
	Type         string
	Id           string
	Data         string
	ResponseChan chan Response
}

type Response struct {
	Type      string
	RequestId string
	Data      interface{}
}

type LocalFileDescription struct {
	FullPath             string
	Name                 string
	IsDir                bool
	IsMedia              bool
	Upload               bool
	Recursive            bool
	Size                 int64
	MediaContained       int64
	DirectoriesContained int64
}

type SettingsData struct {
	UploadImages    bool     `json:"Upload images"`
	UploadVideo     bool     `json:"Upload video"`
	UploadRaw       bool     `json:"Upload RAW"`
	ImageExtensions []string `json:"Image extensions"`
	RawExtensions   []string `json:"RAW extensions"`
	VideoExtensions []string `json:"Video extensions"`
}

func (state *State) AcceptRequestsFromController() {
	log.Println("Accepting requests from controller")
	var wg sync.WaitGroup

	for request := range state.requestChan {
		log.Println("Received control message", request.Type, request.Id)

		switch request.Type {
		case "getDirectoryContents":
			wg.Add(1)
			go state.getDirectoryContents(wg, request)
		case "retryUpload":
			wg.Add(1)
			go state.retryUpload(wg, request)
		case "listSettings":
			wg.Add(1)
			go state.listSettings(wg, request)
		case "getLocalFiles":
			wg.Add(1)
			go state.getLocalFiles(wg, request)
		case "uploadFileOrDirectory":
			wg.Add(1)
			go state.uploadFileOrDirectory(wg, request)
		case "watchAndUploadDirectory":
			wg.Add(1)
			go state.watchAndUploadDirectory(wg, request)
		case "getLocalDirectories":
			wg.Add(1)
			go state.getLocalDirectories(wg, request)
		case "unwatchDirectory":
			wg.Add(1)
			go state.unwatchDirectory(wg, request)
		case "forgetDirectory":
			wg.Add(1)
			go state.forgetDirectory(wg, request)
		default:
			log.Println("Unhandled request tpye")
			request.ResponseChan <- Response{Type: "Error", RequestId: request.Id, Data: fmt.Sprintln("Unable to handle request type: ", request.Type)}
		}
	}
	log.Println("Waiting for controller request processing to complete.")
	wg.Wait()
}

func (s *State) retryUpload(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	sig := r.Data

	//log.Println("in retryUpload for sig", sig)

	existingFile, ok := s.Files[sig]
	if !ok {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: "Could not find file to retry in local library."}
		return
	} else {
		log.Println("Retrying upload file")
		_, _, err := s.RetryFile(existingFile.Path, s.DirectFileChan)
		if err != nil {
			log.Println(err)
		}
	}

	response := Response{Type: "Response", RequestId: r.Id}
	r.ResponseChan <- response
	return
}

func (s *State) getDirectoryContents(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	contains := func(ary []string, value string) bool {
		for _, elem := range ary {
			if elem == value {
				return true
			}
		}
		return false
	}

	isUploadableExtension := func(extension string) bool {
		return contains(s.ImageExtensions, extension) || contains(s.RawExtensions, extension) || contains(s.VideoExtensions, extension)
	}

	path := r.Data
	file, err := os.Open(path)
	if err != nil {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: err.Error()}
		return
	}

	fileInfos, err := file.Readdir(-1)
	if err != nil {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: err.Error()}
		return
	}

	rd := []LocalFileDescription{}
	var fi os.FileInfo
	for _, fi = range fileInfos {
		extension := strings.ToUpper(filepath.Ext(fi.Name()))

		containedPath := filepath.Join(path, fi.Name())
		var uploadDir, recursiveDir bool
		if fi.IsDir() {
			localDir, ok := s.GetDirectory(containedPath)
			//log.Println("Checking for localdir", containedPath)
			if ok {
				//log.Println("Found localdir", containedPath)
				uploadDir = localDir.Upload
				recursiveDir = localDir.Recurisive
			}
		}

		containedFiles, _ := ioutil.ReadDir(containedPath)

		dirCount, mediaCount := 0, 0
		for _, fi2 := range containedFiles {
			extension2 := strings.ToUpper(filepath.Ext(fi2.Name()))
			if fi2.IsDir() {
				dirCount++
				continue
			}
			if isUploadableExtension(extension2) {
				mediaCount++
			}
		}
		rd = append(rd, LocalFileDescription{
			FullPath:             filepath.Join(path, fi.Name()),
			Name:                 fi.Name(),
			IsDir:                fi.IsDir(),
			Size:                 fi.Size(),
			MediaContained:       int64(mediaCount),
			DirectoriesContained: int64(dirCount),
			IsMedia:              isUploadableExtension(extension),
			Upload:               uploadDir,
			Recursive:            recursiveDir,
		})
	}

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = rd
	r.ResponseChan <- response

	return
}

func (s *State) getLocalDirectories(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	rd := []LocalDirectory{}

	for _, localDir := range s.Directories {
		rd = append(rd, localDir)
	}

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = rd
	r.ResponseChan <- response

	return
}

func (s *State) ToSettingsData() SettingsData {
	return SettingsData{
		UploadImages:    s.UploadImages,
		UploadVideo:     s.UploadVideo,
		UploadRaw:       s.UploadRaw,
		ImageExtensions: s.ImageExtensions,
		RawExtensions:   s.RawExtensions,
		VideoExtensions: s.VideoExtensions,
	}
}

func (s *State) listSettings(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = s.ToSettingsData()
	r.ResponseChan <- response

	return
}

func (s *State) getLocalFiles(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	var rd []File

	for _, file := range s.Files {
		rd = append(rd, file)
	}

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = rd
	r.ResponseChan <- response

	return
}

func (s *State) uploadFileOrDirectory(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	path := r.Data

	//log.Println("Received command to upload:", path)

	passedFile, err := os.Open(path)
	if err != nil {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: err.Error()}
		return
	}

	if fileInfo, _ := passedFile.Stat(); fileInfo.IsDir() {
		log.Println("Passed directory")
		go func() {
			log.Println("Trying to upload directory")
			s.UploadDirectory(path, s.DirectFileChan)
			response := Response{Type: "Response", RequestId: r.Id}
			response.Data = "File uploaded"
			r.ResponseChan <- response
		}()
	} else {
		log.Println("Passed file")
		go func() {
			log.Println("Trying to upload file")
			_, _, err := s.UploadFile(path, s.DirectFileChan)
			if err != nil {
				log.Println(err)
			}
			response := Response{Type: "Response", RequestId: r.Id}
			response.Data = "Directory uploaded"
			r.ResponseChan <- response
		}()
	}

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = "Upload command received."
	r.ResponseChan <- response

	return
}

func (s *State) watchAndUploadDirectory(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	path := r.Data

	log.Println("Received command to upload and watch:", path)

	passedFile, err := os.Open(path)
	if err != nil {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: err.Error()}
		return
	}

	if fileInfo, _ := passedFile.Stat(); !fileInfo.IsDir() {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: "Can only watch directories."}
		return
	}

	//log.Println("Passed directory")

	directory, ok := s.GetDirectory(path)
	if !ok {
		directory = LocalDirectory{Path: path}
	}
	directory.MissingOnFilesystem = false
	directory.Recurisive = false
	directory.Upload = true
	s.SetDirectory(directory)
	s.Save()

	log.Println("saved state in watchAndUploadDirectory")

	go func() {
		log.Println("Trying to upload directory")
		s.UploadDirectory(path, s.DirectFileChan)
		response := Response{Type: "Response", RequestId: r.Id}
		response.Data = "Directory uploaded"
		r.ResponseChan <- response
	}()

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = "Upload command received."
	r.ResponseChan <- response

	return
}

func (s *State) unwatchDirectory(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	path := r.Data

	log.Println("Received command to unwatch:", path)

	passedFile, err := os.Open(path)
	if err != nil {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: err.Error()}
		return
	}

	//log.Println("Passed directory")

	directory, ok := s.GetDirectory(path)
	if !ok {
		directory = LocalDirectory{Path: path}
	}

	directory.MissingOnFilesystem = false
	if fileInfo, err := passedFile.Stat(); !fileInfo.IsDir() || err != nil {
		directory.MissingOnFilesystem = true
	}

	directory.Recurisive = false
	directory.Upload = false
	s.SetDirectory(directory)
	s.Save()

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = "Unwatch command received."
	r.ResponseChan <- response

	return
}

func (s *State) forgetDirectory(wg sync.WaitGroup, r Request) {
	defer func() { wg.Done() }()

	path := r.Data

	log.Println("Received command to forget:", path)

	_, existsInDb := s.GetDirectory(path)
	if !existsInDb {
		r.ResponseChan <- Response{Type: "Error", RequestId: r.Id, Data: "Directory not found."}
		return
	}

	s.DelDirectory(path)
	s.Save()

	response := Response{Type: "Response", RequestId: r.Id}
	response.Data = "Forget command received."
	r.ResponseChan <- response

	return
}
