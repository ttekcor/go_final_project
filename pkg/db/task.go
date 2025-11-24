package db

import (
	"database/sql"
	"time"
)

type Task struct {
	ID      int64  `db:"id" json:"id"`
	Date    string `db:"date" json:"date"`
	Title   string `db:"title" json:"title"`
	Comment string `db:"comment" json:"comment"`
	Repeat  string `db:"repeat" json:"repeat"`
}

// Tasks возвращает не более limit задач, отсортированных по дате и id.
func Tasks(limit int) ([]*Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := Handle().Query(`SELECT id, date, title, comment, repeat
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
func GetTask(id int64) (*Task, error) {
	var t Task
	err := Handle().QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, id).
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
func TasksSearch(limit int, search string) ([]*Task, error) {
	if limit <= 0 {
		limit = 50
	}
	// Попытка распознать дату в формате 02.01.2006
	if t, err := time.Parse("02.01.2006", search); err == nil {
		date := t.Format("20060102")
		rows, err := Handle().Query(`SELECT id, date, title, comment, repeat
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
	rows, err := Handle().Query(`SELECT id, date, title, comment, repeat
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

