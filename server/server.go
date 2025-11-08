package server

import (
	"log"
	"main/handlers"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Service struct {
	log    *log.Logger
	server http.Server
	router *http.ServeMux
}

func NewServer() *http.Server {
	logger := log.New(log.Writer(), "SERVER: ", log.LstdFlags)

	r := chi.NewRouter()
	r.Get("/", handlers.HandlerHTML)
	

	server := &http.Server{
		Addr:         ":7540",          // Порт 8080
		Handler:      r,                // HTTP-роутер
		ErrorLog:     logger,           // Логгер
		ReadTimeout:  5 * time.Second,  // Таймаут для чтения - 5 секунд
		WriteTimeout: 10 * time.Second, // Таймаут для записи - 10 секунд
		IdleTimeout:  15 * time.Second, // Таймаут ожидания - 15 секунд
	}

	return server
}
func main() {

	_ = NewServer()

}
