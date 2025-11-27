package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"main/pkg/db"
	"main/pkg/scheduler"
)

const dateFormat = "20060102"

var (
	ErrNotFound      = errors.New("задача не найдена")
	ErrTitleRequired = errors.New("title: обязателен")
	ErrBadDate       = errors.New("date: неверный формат")
	ErrBadRepeat     = errors.New("repeat: неверный формат")
)

// TaskService предоставляет бизнес-логику для работы с задачами
type TaskService struct {
	store *db.TaskStore
}

// NewTaskService создает новый экземпляр TaskService
func NewTaskService(store *db.TaskStore) *TaskService {
	return &TaskService{store: store}
}

// GetTasks возвращает список задач с учётом поиска и лимита.
func (s *TaskService) GetTasks(search string, limit int) ([]*db.Task, error) {
	if search == "" {
		return s.store.Tasks(limit)
	}
	return s.store.TasksSearch(limit, search)
}

// GetTask возвращает задачу по id.
func (s *TaskService) GetTask(id int64) (*db.Task, error) {
	t, err := s.store.GetTask(id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// AddTask добавляет новую задачу.
func (s *TaskService) AddTask(now time.Time, date, title, comment, repeat string) (int64, error) {
	title = trim(title)
	if title == "" {
		return 0, ErrTitleRequired
	}

	today := now.Format(dateFormat)
	if date == "" {
		date = today
	}
	if _, err := time.Parse(dateFormat, date); err != nil {
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
	return s.store.AddTask(date, title, comment, repeat)
}

// UpdateTask изменяет существующую задачу.
func (s *TaskService) UpdateTask(now time.Time, id int64, date, title, comment, repeat string) error {
	// Проверим наличие
	if _, err := s.GetTask(id); err != nil {
		return err
	}
	title = trim(title)
	if title == "" {
		return ErrTitleRequired
	}
	today := now.Format(dateFormat)
	if date == "" {
		date = today
	}
	if _, err := time.Parse(dateFormat, date); err != nil {
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
	return s.store.UpdateTask(id, date, title, comment, repeat)
}

// DeleteTask удаляет задачу по id.
func (s *TaskService) DeleteTask(id int64) error {
	n, err := s.store.DeleteTask(id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DoneTask помечает задачу выполненной: удаляет, либо переносит дату согласно repeat.
func (s *TaskService) DoneTask(id int64) error {
	t, err := s.store.GetTask(id)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if trim(t.Repeat) == "" {
		n, err := s.store.DeleteTask(id)
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	}
	prevDate, err := time.Parse(dateFormat, t.Date)
	if err != nil {
		return ErrBadDate
	}
	next, err := scheduler.NextDate(prevDate, t.Date, t.Repeat)
	if err != nil {
		return err
	}
	return s.store.UpdateTaskDate(id, next)
}

func trim(s string) string {
	return strings.TrimSpace(s)
}
