package local

import (
	"../util"
	"github.com/howeyc/fsnotify"
	"log"
)

type Watcher struct {
	s        *State
	DoneChan chan int
	path     string
	watching bool
}

func (s *State) WatchFilesystem(path string) {
	log.Println("Making watcher for directory:", path)
	w := NewWatcher(s, path)
	log.Println("Starting watcher for directory:", path)
	w.Start()
	s.watchers[path] = w
	log.Println("Started watching directory:", path)
}

func NewWatcher(s *State, path string) Watcher {
	w := Watcher{
		s:        s,
		path:     path,
		watching: false,
	}
	w.DoneChan = make(chan int)

	return w
}

func (w *Watcher) Path() string {
	return w.path
}

func (w *Watcher) Start() {
	log.Println("Watching directory", w.path)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
		return
	}

	// Process events
	go func() {
		defer func() {
			watcher.Close()
			//log.Println("Closed watcher")
		}()
		w.watching = true
		for {
			select {
			case ev := <-watcher.Event:
				log.Println("filesystem event:", ev)
				if ev.IsCreate() || ev.IsModify() {
					path := ev.Name
					signature := util.CalculateSignature(path)
					file := File{
						Signature: signature,
						Path:      path,
					}
					w.s.WatchFileChan <- file
				}
			case err := <-watcher.Error:
				log.Println("error:", err)
			case <-w.DoneChan:
				log.Println("Watcher received term signal")
				w.watching = false
				return
			}
		}
	}()

	err = watcher.Watch(w.path)
	if err != nil {
		log.Println(err)
		return
	}
}

func (w *Watcher) Stop() {
	// nonblocking send
	select {
	case w.DoneChan <- 1:
	default:
	}
}
