package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

const DefaultDBFile = "scheduler.db"

// глобальный дескриптор БД
var db *sql.DB

// SQL схема для первичной установки
const schema = `
CREATE TABLE IF NOT EXISTS scheduler (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date CHAR(8) NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	comment TEXT NOT NULL DEFAULT '',
	repeat TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler(date);
`

// Init открывает БД с помощью database/sql и при необходимости создаёт таблицу/индекс
func Init(dbFile string) error {
	if dbFile == "" {
		dbFile = DefaultDBFile
	}

	_, statErr := os.Stat(dbFile)
	install := statErr != nil

	handle, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	// Проверяем соединение
	if err := handle.Ping(); err != nil {
		_ = handle.Close()
		return fmt.Errorf("ping sqlite: %w", err)
	}

	db = handle

	if install {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("install schema: %w", err)
		}
	}
	return nil
}
