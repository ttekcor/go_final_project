package server

import (
	"log"
	"main/pkg/api"
	"main/pkg/config"
	"main/pkg/db"
	"main/pkg/service"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func NewServer(cfg *config.Config, taskStore *db.TaskStore) *http.Server {
	taskService := service.NewTaskService(taskStore)
	logger := log.New(log.Writer(), "SERVER: ", log.LstdFlags)

	r := chi.NewRouter()
	r.Get("/", api.HandlerHTML)

	// API
	r.Route("/api", func(r chi.Router) {
		r.Get("/nextdate", api.HandlerNextDate)
		r.Get("/tasks", api.HandlerTasks(taskService))
		r.Get("/task", api.HandlerGetTask(taskService))
		r.Post("/task", api.HandlerAddTask(taskService))
		r.Put("/task", api.Auth(api.HandlerEditTask(taskService), cfg))
		r.Delete("/task", api.Auth(api.HandlerDeleteTask(taskService), cfg))
		r.Post("/task/done", api.Auth(api.HandlerDone(taskService), cfg))
		r.Post("/signin", api.HandlerSignIn(cfg))
	})

	// Раздача статических файлов из каталога ./web
	staticFs := http.FileServer(http.Dir("./web"))
	r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir("./web/js"))))
	r.Handle("/css/*", http.StripPrefix("/css/", http.FileServer(http.Dir("./web/css"))))
	r.Handle("/favicon.ico", staticFs)
	r.Handle("/*", staticFs)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return server
}
