package main

import (
	"log"
	"main/server"
)

func main() {
	logger := log.New(log.Writer(), "MAIN: ", log.LstdFlags)
	server := server.NewServer()
	logger.Printf("Сервер запускается на порту %s...", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Ошибка запуска сервера: ", err)
	}
}
