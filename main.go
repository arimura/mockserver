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

	endpointInfos := makeEndpointInfos(*dataPath)

	watch(*dataPath)

	info("start server on port " + *port)
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
	mux.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		info(fmt.Sprintf("%s %s %s %s", r.Method, r.URL, r.Proto, r.UserAgent()))

		writer.WriteHeader(http.StatusNotFound)
		writer.Write([]byte("404"))
	})

	for _, endpointInfo := range endpointInfos {
		mux.HandleFunc(endpointInfo.urlPath, func(w http.ResponseWriter, r *http.Request) {
			data, error := ioutil.ReadFile(endpointInfo.filePath)
			if error != nil {
				die(fmt.Sprintf("file doesn't exist: %s", endpointInfo.filePath))
			}
			info(fmt.Sprintf("%s %s %s %s", r.Method, r.URL, r.Proto, r.UserAgent()))

			time.Sleep(time.Duration(delay) * time.Millisecond)
			dump, _ := ioutil.ReadAll(r.Body)

			var prettyJSON bytes.Buffer
			error = json.Indent(&prettyJSON, dump, "", "  ")
			if error == nil {
				info(string(prettyJSON.Bytes()))
			} else {
				info(string(dump))
			}
			w.Header().Set("Content-Type", http.DetectContentType(data))
			fmt.Fprint(w, string(data))
		})
	}
}

func makeEndpointInfos(dirPath string) []endpointInfo {
	fi, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		die(fmt.Sprintf("not exists: %s", dirPath))
	}
	if !fi.Mode().IsDir() {
		die(fmt.Sprintf("not dir: %s", dirPath))
	}

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

func info(v string) {
	log.Println(v)
}

func die(v string) {
	os.Stderr.WriteString(v + "\n")
	os.Exit(1)
}
