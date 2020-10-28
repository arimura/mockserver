package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

var cachedResponses map[string][]byte

func main() {
	dataPath := flag.String("data", "./data", "specify response dir")
	port := flag.String("port", "8080", "specify port")
	delay := flag.Int64("delay", 0, "mille sec delay for response")
	flag.Parse()

	fi, err := os.Stat(*dataPath)
	if os.IsNotExist(err) || !fi.Mode().IsDir() {
		log.Fatalf("No dir: %s", *dataPath)
	}

	cachedResponses = make(map[string][]byte)

	watch(*dataPath)

	log.Printf("start server on port %s", *port)
	s := &server{
		mux:      http.NewServeMux(),
		dataPath: *dataPath,
		port:     *port,
		delay:    *delay,
	}
	s.run()
}

type server struct {
	dataPath string
	port     string
	delay    int64
	mux      *http.ServeMux
}

func (s *server) run() {
	s.registerEndpoints()
	http.ListenAndServe(":"+s.port, s.mux)
}

func (s *server) registerEndpoints() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s %s\n", r.Method, r.URL, r.Proto, r.UserAgent())

		s.printBody(r.Body)

		time.Sleep(time.Duration(s.delay) * time.Millisecond)

		filePath := s.dataPath + "/" + r.URL.Path[1:]

		cachedResponse := cachedResponses[filePath]
		if cachedResponse != nil {
			w.Header().Set("Content-Type", http.DetectContentType(cachedResponse))
			fmt.Fprint(w, string(cachedResponse))
			return
		}

		data, error := ioutil.ReadFile(filePath)
		if error != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404"))
			return
		}

		cachedResponses[filePath] = data

		w.Header().Set("Content-Type", http.DetectContentType(data))
		fmt.Fprint(w, string(data))
	})
}

func (s *server) printBody(r io.Reader) {
	dump, _ := ioutil.ReadAll(r)
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, dump, "", "  ")
	if error == nil {
		log.Printf("request body: %s\n", prettyJSON.Bytes())
	} else {
		log.Printf("request body: %s\n", string(dump))
	}
}

func watch(dataPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Printf("%s modified. Cache is cleared", event.Name)
					cachedResponses[event.Name] = nil
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	log.Println("start watching:", dataPath)
	err = watcher.Add(dataPath)
	if err != nil {
		log.Fatal(err)
	}
}
