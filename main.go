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
	"net/url"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	dataPath := flag.String("data", "./data", "specify response dir")
	port := flag.String("port", "8080", "specify port")
	delay := flag.Int64("delay", 0, "mille sec delay for response")
	requestQueryUnescape := flag.Bool("requestQueryUnescape", true, "unescape query of request in log")
	flag.Parse()

	fi, err := os.Stat(*dataPath)
	if os.IsNotExist(err) || !fi.Mode().IsDir() {
		log.Fatalf("No dir: %s", *dataPath)
	}

	log.Printf("start server on port %s", *port)
	s := &server{
		mux:                  http.NewServeMux(),
		dataPath:             *dataPath,
		port:                 *port,
		delay:                *delay,
		cachedResponses:      make(map[string][]byte),
		requestQueryUnescape: *requestQueryUnescape,
	}
	s.run()
}

type server struct {
	dataPath             string
	port                 string
	delay                int64
	mux                  *http.ServeMux
	cachedResponses      map[string][]byte
	requestQueryUnescape bool
}

func (s *server) run() {
	s.watch()
	s.registerEndpoints()
	http.ListenAndServe(":"+s.port, s.mux)
}

func (s *server) registerEndpoints() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var u string
		if s.requestQueryUnescape {
			u, _ = url.QueryUnescape(r.URL.String())
		} else {
			u = r.URL.String()
		}

		log.Printf("%s %s %s %s\n", r.Method, u, r.Proto, r.UserAgent())

		s.printBody(r.Body)

		time.Sleep(time.Duration(s.delay) * time.Millisecond)

		filePath := s.dataPath + "/" + r.URL.Path[1:]

		cachedResponse := s.cachedResponses[filePath]
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

		s.cachedResponses[filePath] = data

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

func (s *server) watch() {
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
					s.cachedResponses[event.Name] = nil
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	log.Println("start watching:", s.dataPath)
	err = watcher.Add(s.dataPath)
	if err != nil {
		log.Fatal(err)
	}
}
