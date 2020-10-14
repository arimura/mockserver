package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type endpointInfo struct {
	urlPath  string
	filePath string
}

func main() {
	dataPath := flag.String("data", "./data", "specify response dir")
	port := flag.String("port", "8080", "specify port")
	delay := flag.Int64("delay", 0, "mille sec delay for response")
	flag.Parse()

	fi, err := os.Stat(*dataPath)
	if os.IsNotExist(err) || !fi.Mode().IsDir() {
		log.Fatalf("No dir: %s", *dataPath)
	}

	endpointInfos := makeEndpointInfos(*dataPath)

	watch(*dataPath)

	log.Printf("start server on port %s", *port)
	mux := http.NewServeMux()
	registerEndpoints(mux, endpointInfos, *delay)
	http.ListenAndServe(":"+*port, mux)
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
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Panicln("modified file:", event.Name)
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

func registerEndpoints(mux *http.ServeMux, endpointInfos []endpointInfo, delay int64) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s %s\n", r.Method, r.URL, r.Proto, r.UserAgent())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
	})

	for _, endpointInfo := range endpointInfos {
		//bind local var
		ei := endpointInfo
		mux.HandleFunc(ei.urlPath, func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s %s %s\n", r.Method, r.URL, r.Proto, r.UserAgent())

			data, error := ioutil.ReadFile(ei.filePath)
			if error != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("404"))
				return
			}

			time.Sleep(time.Duration(delay) * time.Millisecond)
			dump, _ := ioutil.ReadAll(r.Body)

			var prettyJSON bytes.Buffer
			error = json.Indent(&prettyJSON, dump, "", "  ")
			if error == nil {
				log.Printf("request body: %s\n", prettyJSON.Bytes())
			} else {
				log.Printf("request body: %s\n", string(dump))
			}
			w.Header().Set("Content-Type", http.DetectContentType(data))
			fmt.Fprint(w, string(data))
		})
	}
}

func makeEndpointInfos(dirPath string) []endpointInfo {
	if strings.HasPrefix(dirPath, "./") {
		dirPath = dirPath[2:]
	}

	var endpointInfos []endpointInfo
	filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		urlPath := strings.Replace(filePath, dirPath, "", 1)
		urlPath = strings.Replace(urlPath, "__S__", "/", -1)

		endpointInfos = append(endpointInfos, endpointInfo{
			urlPath,
			filePath,
		})
		return nil
	})

	return endpointInfos
}
