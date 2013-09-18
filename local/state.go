package local

import (
	"../api"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
)

type LocalDirectory struct {
	Path                string
	Recurisive          bool
	Upload              bool
	MissingOnFilesystem bool
	UpdatedAt           time.Time
}

type File struct {
	Signature           string
	Path                string
	PendingMediaId      string
	MediaId             string
	UploadedAt          time.Time
	UpdatedAt           time.Time
	Status              string
	MissingOnFilesystem bool
	Name                string
	Extension           string
}

func (f *File) ToJson() string {
	b, err := json.Marshal(f)
	if err != nil {
		log.Println("Could not JSON encode file:", err)
		return "null"
	}
	return string(b)
}

type State struct {
	Files           map[string]File `json:"Files"`
	Api             api.API
	StateFile       string
	MaxUploadsChan  chan int  `json:"-"`
	WatchFileChan   chan File `json:"-"`
	DirectFileChan  chan File `json:"-"`
	DoneChan        chan int  `json:"-"`
	ImageExtensions []string
	RawExtensions   []string
	VideoExtensions []string
	UploadImages    bool
	UploadVideo     bool
	UploadRaw       bool
	observerChan    chan Response `json:"-"`
	requestChan     chan Request  `json:"-"`
	Directories     map[string]LocalDirectory
	watchers        map[string]Watcher
}

func NewState(path string) State {
	var ns State
	ns.Files = make(map[string]File)
	ns.StateFile = path
	ns.UploadImages = true
	ns.UploadVideo = true
	ns.UploadRaw = true
	ns.ImageExtensions = []string{".JPG", ".JPEG", ".PNG"}
	ns.RawExtensions = []string{".NEF", ".CR2"}
	ns.VideoExtensions = []string{".MOV"}
	ns.observerChan = nil
	ns.requestChan = nil
	ns.watchers = make(map[string]Watcher)
	ns.Directories = make(map[string]LocalDirectory)
	return ns
}

func (state *State) RegisterObserver(newObserverChan chan Response) {
	state.observerChan = newObserverChan
	log.Println("Registered state observer")
}

func (state *State) RegisterController(newrequestChan chan Request) {
	state.requestChan = newrequestChan
	log.Println("Registered state controller")
	go state.AcceptRequestsFromController()
}

func (state *State) logEvent(event Response) {
	//log.Println("Logging state event")
	if state.observerChan == nil {
		return
	}
	go func() {
		//log.Println("Actually logging state event")
		state.observerChan <- event
	}()
}

func (state *State) SetDirectory(d LocalDirectory) {
	d.UpdatedAt = time.Now()
	state.Directories[d.Path] = d
	state.logEvent(Response{Type: "directoryUpdate", RequestId: "", Data: d.Path})
	state.UpdateDirectoryWatchers()
}

func (state *State) GetDirectory(path string) (savedDirectory LocalDirectory, ok bool) {
	savedDirectory, ok = state.Directories[path]
	return
}

func (state *State) DelDirectory(path string) {
	_, present := state.Directories[path]
	delete(state.Directories, path)
	if present {
		state.logEvent(Response{Type: "directoryDelete", RequestId: "", Data: path})
	}
	return
}

func (state *State) UpdateDirectoryWatchers() {
	//log.Println("UpdateDirectoryWatchers")
	for _, watcher := range state.watchers {
		//log.Println("Stopping watcher", watcher)
		watcher.Stop()
	}
	for path, localDir := range state.Directories {
		if localDir.Upload {
			state.WatchFilesystem(path)
		} else {
			//log.Println("No watching dir: ", path)
		}
	}
	//log.Println("end UpdateDirectoryWatchers")
}

func (state *State) UploadWatchedDirectories() {
	for path, localDir := range state.Directories {
		if localDir.Upload {
			state.UploadDirectory(path, state.DirectFileChan)
		}
	}
}

func (state *State) SetFile(file File) {
	file.UpdatedAt = time.Now()
	state.Files[file.Signature] = file
	state.logEvent(Response{Type: "fileUpdate", RequestId: "", Data: file.Signature})
	//log.Println("saved file", file.Signature)
	//log.Println("total in db", len(state.Files))
}

func (state *State) GetFile(sig string) (savedFile File, ok bool) {
	savedFile, ok = state.Files[sig]
	//log.Printf("%v", state.Files[sig].Signature)
	//log.Println("file from db", savedFile)
	return
}

func (state *State) Save() {
	jsonBytes, err := json.Marshal(state)
	if err != nil {
		log.Println("Could not serialize Files database", err)
		panic("Could not save state file")
	}
	err = ioutil.WriteFile(state.StateFile, jsonBytes, 0660)
	if err != nil {
		log.Println("Could not open file for writing:", err)
		panic("Could not save state file")
	}
	//log.Println("Saved state file")
}

func (state *State) Load() {
	file, e := ioutil.ReadFile(state.StateFile)
	if e != nil {
		//log.Printf("File error: %v\n", e)
		log.Println("Could not load state file.")
	}
	//log.Printf("%s\n", string(file))

	//var parsed map[string]File
	json.Unmarshal(file, state)
	//log.Printf("Results: %v\n", state.files)
}
