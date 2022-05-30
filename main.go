package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/atekoa/dvc-http-remote/pkg/handler"
	"github.com/atekoa/dvc-http-remote/pkg/storage"

	_ "net/http/pprof"
)

func NewTempDir() (string, func()) {
	abs, _ := filepath.Abs("remote-folder")
	if err := os.MkdirAll("remote-folder", 0777); err != nil {
		panic(err)
	}
	return "remote-folder", func() { os.RemoveAll(abs) }
}

func main() {
	log.Printf("I am %s", os.Getenv("HOSTNAME"))

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	dir, cleanup := NewTempDir()
	defer cleanup()

	storage := storage.NewStorageSiteLoader(dir)

	pathPrefix := os.Getenv("PATH_PREFIX")

	r := mux.NewRouter()

	handler.Attach(
		r,
		pathPrefix,
		storage,
	)

	server := http.Server{
		Addr:              ":8080",
		Handler:           handlers.LoggingHandler(os.Stderr, r),
		ReadTimeout:       1 * time.Hour,
		WriteTimeout:      1 * time.Hour,
		IdleTimeout:       1 * time.Hour,
		ReadHeaderTimeout: 1 * time.Hour,
	}

	log.
		WithField("path prefix", pathPrefix).
		Info("ready")

	go runProfiler()
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func runProfiler() {
	log.Println(http.ListenAndServe(":7777", nil))
}

func envMustBeSet(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Panicf("variable %s is not defined", key)
	}
	return value
}
