package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

const DefaultDBFile = "scheduler.db"

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
// Возвращает *sql.DB для использования через dependency injection
func Init(dbFile string) (*sql.DB, error) {
	if dbFile == "" {
		dbFile = DefaultDBFile
	}

	_, statErr := os.Stat(dbFile)
	install := statErr != nil

	handle, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Проверяем соединение
	if err := handle.Ping(); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if install {
		if _, err := handle.Exec(schema); err != nil {
			_ = handle.Close()
			return nil, fmt.Errorf("install schema: %w", err)
		}
	}
	return handle, nil
}
