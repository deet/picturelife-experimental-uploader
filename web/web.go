package web

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/cratonica/trayhost"
	"github.com/deet/picturelife-experimental-uploader/local"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
)

type connection struct {
	// The websocket connection.
	ws    *websocket.Conn
	state *local.State

	// Buffered channel of outbound messages.
	send chan outgoingMessage
}

type outgoingMessage struct {
	Type      string
	RequestId string
	Data      interface{}
}

type incomingRequest struct {
	Type      string
	Data      string
	RequestId string
}

func (f *outgoingMessage) ToJson() string {
	b, err := json.Marshal(f)
	if err != nil {
		log.Println("Could not JSON encode websocket message:", err)
		return "null"
	}
	return string(b)
}

func (c *connection) reader(appState *local.State) {
	requestChan := make(chan local.Request, 1000)
	appState.RegisterController(requestChan)
	var wg sync.WaitGroup

	for {
		var message incomingRequest
		err := websocket.JSON.Receive(c.ws, &message)
		if err != nil {
			log.Println("Could not parse incoming websocket message:", err)
			break
		}
		//log.Println("Received websocket message", message.Type, message.Data, message.RequestId)
		wg.Add(1)
		go func() {
			defer wg.Done()
			responseChan := make(chan local.Response)
			requestChan <- local.Request{Type: message.Type, Data: message.Data, ResponseChan: responseChan}
			response := <-responseChan
			c.send <- outgoingMessage{Type: response.Type, Data: response.Data, RequestId: message.RequestId}
			return
		}()
	}
	log.Println("Controller waiting for response to requests to state.")
	wg.Wait()
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := websocket.JSON.Send(c.ws, &message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func observeState(appState *local.State, c *connection) {
	observerChan := make(chan local.Response, 1000)
	appState.RegisterObserver(observerChan)
	for event := range observerChan {
		//log.Println("Received state event")
		switch event.Type {
		case "fileUpdate":
			file, ok := appState.GetFile(string(event.Data.(string)))
			if ok {
				rd := []local.File{file}
				c.send <- outgoingMessage{Type: "FileUpdate", Data: rd}
			}
		case "directoryUpdate":
			dir, ok := appState.GetDirectory(string(event.Data.(string)))
			if ok {
				rd := []local.LocalDirectory{dir}
				c.send <- outgoingMessage{Type: "DirectoryUpdate", Data: rd}
			}
		case "directoryDelete":
			c.send <- outgoingMessage{Type: "DirectoryDelete", Data: event.Data.(string)}
		}
	}
}

func StartWebUi(appState *local.State) {
	wd, _ := os.Getwd()
	base := path.Dir(wd)
	final := path.Join(base, "web", "assets")
	log.Println("Serving directory:", final)
	//http.ListenAndServe(":7111", http.FileServer(http.Dir("web/assets")))

	http.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		c := &connection{send: make(chan outgoingMessage, 256), ws: ws}
		//go sendFiles(appState, c)
		go observeState(appState, c)
		go c.writer()
		c.reader(appState)
	}))
	http.Handle("/", http.FileServer(http.Dir("web/assets")))

	trayhost.SetUrl("http://localhost:7111")

	http.ListenAndServe(":7111", nil)

}
