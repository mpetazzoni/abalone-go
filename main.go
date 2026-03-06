package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/mpetazzoni/abalone-go/server"
)

//go:embed web
var webContent embed.FS

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	// Strip the "web" prefix from the embedded filesystem
	webFS, err := fs.Sub(webContent, "web")
	if err != nil {
		log.Fatal(err)
	}

	srv := server.NewServer(http.FS(webFS))
	log.Fatal(srv.ListenAndServe(*addr))
}
