package server

import (
	"log"
	"main/pkg/api"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

func NewServer() *http.Server {
	logger := log.New(log.Writer(), "SERVER: ", log.LstdFlags)

	r := chi.NewRouter()
	r.Get("/", api.HandlerHTML)

	// Раздача статических файлов из каталога ./web
	staticFs := http.StripPrefix("/", http.FileServer(http.Dir("./web")))
	r.Handle("/*", staticFs)

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return server
}

