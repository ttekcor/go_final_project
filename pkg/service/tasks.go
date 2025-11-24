package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"main/pkg/db"
	"main/pkg/scheduler"
)

var (
	ErrNotFound      = errors.New("задача не найдена")
	ErrTitleRequired = errors.New("title: обязателен")
	ErrBadDate       = errors.New("date: неверный формат")
	ErrBadRepeat     = errors.New("repeat: неверный формат")
)

// GetTasks возвращает список задач с учётом поиска и лимита.
func GetTasks(search string, limit int) ([]*db.Task, error) {
	if search == "" {
		return db.Tasks(limit)
	}
	return db.TasksSearch(limit, search)
}

// GetTask возвращает задачу по id.
func GetTask(id int64) (*db.Task, error) {
	t, err := db.GetTask(id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// AddTask добавляет новую задачу.
func AddTask(now time.Time, date, title, comment, repeat string) (int64, error) {
	title = trim(title)
	if title == "" {
		return 0, ErrTitleRequired
	}

	today := now.Format(`20060102`)
	if date == "" {
		date = today
	}
	if _, err := time.Parse("20060102", date); err != nil {
		return 0, ErrBadDate
	}
	if date < today {
		date = today
	}
	if repeat != "" {
		if _, err := scheduler.NextDate(now, date, repeat); err != nil {
			return 0, ErrBadRepeat
		}
	}
	res, err := db.Handle().Exec(`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
		date, title, comment, repeat)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTask изменяет существующую задачу.
func UpdateTask(now time.Time, id int64, date, title, comment, repeat string) error {
	// Проверим наличие
	if _, err := GetTask(id); err != nil {
		return err
	}
	title = trim(title)
	if title == "" {
		return ErrTitleRequired
	}
	today := now.Format(`20060102`)
	if date == "" {
		date = today
	}
	if _, err := time.Parse("20060102", date); err != nil {
		return ErrBadDate
	}
	if date < today {
		date = today
	}
	if repeat != "" {
		if _, err := scheduler.NextDate(now, date, repeat); err != nil {
			return ErrBadRepeat
		}
	}
	_, err := db.Handle().Exec(`UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?`,
		date, title, comment, repeat, id)
	return err
}

// DeleteTask удаляет задачу по id.
func DeleteTask(id int64) error {
	res, err := db.Handle().Exec(`DELETE FROM scheduler WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DoneTask помечает задачу выполненной: удаляет, либо переносит дату согласно repeat.
func DoneTask(id int64) error {
	t, err := GetTask(id)
	if err != nil {
		return err
	}
	if trim(t.Repeat) == "" {
		return DeleteTask(id)
	}
	prevDate, err := time.Parse("20060102", t.Date)
	if err != nil {
		return ErrBadDate
	}
	next, err := scheduler.NextDate(prevDate, t.Date, t.Repeat)
	if err != nil {
		return err
	}
	_, err = db.Handle().Exec(`UPDATE scheduler SET date=? WHERE id=?`, next, id)
	return err
}

func trim(s string) string {
	return strings.TrimSpace(s)
}


