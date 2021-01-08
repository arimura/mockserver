package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/arimura/mockserver"
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
	s := &mockserver.Server{
		Mux:                  http.NewServeMux(),
		DataPath:             *dataPath,
		Port:                 *port,
		Delay:                *delay,
		CachedResponses:      make(map[string][]byte),
		RequestQueryUnescape: *requestQueryUnescape,
	}
	s.Run()
}
