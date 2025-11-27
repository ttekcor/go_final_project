package db

import (
	"database/sql"
	"time"
)

const (
	dateFormat       = "20060102"
	DefaultTaskLimit = 50
)

type Task struct {
	ID      int64  `db:"id" json:"id"`
	Date    string `db:"date" json:"date"`
	Title   string `db:"title" json:"title"`
	Comment string `db:"comment" json:"comment"`
	Repeat  string `db:"repeat" json:"repeat"`
}

// TaskStore хранит подключение к БД и предоставляет методы для работы с задачами
type TaskStore struct {
	db *sql.DB
}

// NewTaskStore создает новый экземпляр TaskStore
func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

// Tasks возвращает не более limit задач, отсортированных по дате и id.
func (s *TaskStore) Tasks(limit int) ([]*Task, error) {
	if limit <= 0 {
		limit = DefaultTaskLimit
	}
	rows, err := s.db.Query(`SELECT id, date, title, comment, repeat
		FROM scheduler ORDER BY date, id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, err
		}
		list = append(list, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if list == nil {
		list = make([]*Task, 0)
	}
	return list, nil
}

// GetTask возвращает задачу по id.
func (s *TaskStore) GetTask(id int64) (*Task, error) {
	var t Task
	err := s.db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, id).
		Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TasksSearch выполняет выборку по подстроке в title/comment (LIKE) либо по точной дате,
// если search имеет формат 02.01.2006. Возвращает не более limit записей.
func (s *TaskStore) TasksSearch(limit int, search string) ([]*Task, error) {
	if limit <= 0 {
		limit = DefaultTaskLimit
	}
	// Попытка распознать дату в формате 02.01.2006
	if t, err := time.Parse("02.01.2006", search); err == nil {
		date := t.Format(dateFormat)
		rows, err := s.db.Query(`SELECT id, date, title, comment, repeat
			FROM scheduler WHERE date = ? ORDER BY date, id LIMIT ?`, date, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []*Task
		for rows.Next() {
			var t Task
			if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
				return nil, err
			}
			list = append(list, &t)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if list == nil {
			list = make([]*Task, 0)
		}
		return list, nil
	}

	// Поиск по подстроке: регистронезависимо через lower()
	pat := "%" + search + "%"
	rows, err := s.db.Query(`SELECT id, date, title, comment, repeat
		FROM scheduler
		WHERE lower(title) LIKE lower(?) OR lower(comment) LIKE lower(?)
		ORDER BY date, id LIMIT ?`, pat, pat, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, err
		}
		list = append(list, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if list == nil {
		list = make([]*Task, 0)
	}
	return list, nil
}

// AddTask добавляет новую задачу в БД
func (s *TaskStore) AddTask(date, title, comment, repeat string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
		date, title, comment, repeat)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTask обновляет задачу в БД
func (s *TaskStore) UpdateTask(id int64, date, title, comment, repeat string) error {
	_, err := s.db.Exec(`UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?`,
		date, title, comment, repeat, id)
	return err
}

// DeleteTask удаляет задачу из БД
func (s *TaskStore) DeleteTask(id int64) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM scheduler WHERE id=?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// UpdateTaskDate обновляет только дату задачи
func (s *TaskStore) UpdateTaskDate(id int64, date string) error {
	_, err := s.db.Exec(`UPDATE scheduler SET date=? WHERE id=?`, date, id)
	return err
}
