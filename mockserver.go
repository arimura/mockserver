package mockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Server struct {
	DataPath             string
	Port                 string
	Delay                int64
	Mux                  *http.ServeMux
	CachedResponses      map[string][]byte
	RequestQueryUnescape bool
	MacroExpand          bool
}

func (s *Server) Run() {
	s.watch()
	s.registerEndpoints()
	http.ListenAndServe(":"+s.Port, s.Mux)
}

func (s *Server) registerEndpoints() {
	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var u string
		if s.RequestQueryUnescape {
			u, _ = url.QueryUnescape(r.URL.String())
		} else {
			u = r.URL.String()
		}

		log.Printf("%s %s %s %s\n", r.Method, u, r.Proto, r.UserAgent())

		body, bodyError := ioutil.ReadAll(r.Body)
		if bodyError == nil && body != nil {
			s.printAsJSON(body)
		}

		time.Sleep(time.Duration(s.Delay) * time.Millisecond)

		filePath := s.DataPath + "/" + r.URL.Path[1:]

		if !s.MacroExpand {
			cachedResponse := s.CachedResponses[filePath]
			if cachedResponse != nil {
				w.Header().Set("Content-Type", http.DetectContentType(cachedResponse))
				fmt.Fprint(w, string(cachedResponse))
				return
			}
		}

		data, error := ioutil.ReadFile(filePath)
		if error != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404"))
			return
		}

		if !s.MacroExpand {
			s.CachedResponses[filePath] = data
		}

		res := string(data)
		if s.MacroExpand {
			var vals interface{}
			e := json.Unmarshal(body, &vals)
			if e == nil {
				template, templateError := template.New("t").Parse(res)
				if templateError == nil {
					template.Execute(w, vals)
					return
				}
			}
		}

		w.Header().Set("Content-Type", http.DetectContentType(data))
		fmt.Fprint(w, res)
	})
}

func (s *Server) printAsJSON(body []byte) {
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, body, "", "  ")
	if error == nil {
		log.Printf("request body: %s\n", prettyJSON.Bytes())
	} else {
		log.Printf("request body: %s\n", string(body))
	}
}

func (s *Server) watch() {
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
					s.CachedResponses[event.Name] = nil
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	log.Println("start watching:", s.DataPath)
	err = watcher.Add(s.DataPath)
	if err != nil {
		log.Fatal(err)
	}
}
