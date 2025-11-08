package main

import (
	"context"
	"log"
	"main/pkg/db"
	"main/pkg/server"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	logger := log.New(log.Writer(), "MAIN: ", log.LstdFlags)
	// Загружаем переменные окружения из файлов, если есть
	_ = godotenv.Load("tests/.env", "tests\\.env", ".env")
	if err := db.Init("scheduler.db"); err != nil {
		logger.Fatal("Ошибка инициализации БД: ", err)
	}
	srv := server.NewServer()
	logger.Printf("Сервер запускается на порту %s...", srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Ошибка запуска сервера: ", err)
		}
	}()

	// Ожидаем сигнал завершения (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Плавная остановка с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
