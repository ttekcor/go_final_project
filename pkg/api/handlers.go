package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"main/pkg/api/interfaces"
	"main/pkg/config"
	"main/pkg/db"
	"main/pkg/scheduler"
	"main/pkg/service"

	"github.com/golang-jwt/jwt/v5"
)

const dateFormat = "20060102"

func HandlerHTML(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Ошибка чтения файла", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(content); err != nil {
		http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
		return
	}
}

// writeJSON — утилита для JSON-ответов
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, interfaces.ErrorResponse{Error: msg})
}

// HandlerNextDate GET /api/nextdate?now=YYYYMMDD&date=YYYYMMDD&repeat=...
func HandlerNextDate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	nowStr := strings.TrimSpace(q.Get("now"))
	dstart := strings.TrimSpace(q.Get("date"))
	repeat := strings.TrimSpace(q.Get("repeat"))

	var now time.Time
	var err error
	if nowStr == "" {
		now = time.Now()
	} else {
		now, err = time.Parse(dateFormat, nowStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "now: некорректная дата")
			return
		}
	}
	next, err := scheduler.NextDate(now, dstart, repeat)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(next))
}

func Auth(next http.HandlerFunc, cfg *config.Config) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// смотрим наличие пароля
		pass := cfg.Password
		if len(pass) > 0 {
			var jwtToken string // JWT-токен из куки
			// получаем куку
			cookie, err := r.Cookie("token")
			if err == nil {
				jwtToken = cookie.Value
			}
			var valid bool = false
			if strings.TrimSpace(jwtToken) != "" {
				claims := &jwt.RegisteredClaims{}
				token, err := jwt.ParseWithClaims(
					jwtToken,
					claims,
					func(t *jwt.Token) (any, error) {
						// Разрешаем только HS256
						if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
							return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
						}
						return []byte(pass), nil
					},
				)
				if err == nil && token != nil && token.Valid {
					// Дополнительно проверим exp, если он присутствует
					if claims.ExpiresAt == nil || claims.ExpiresAt.Time.After(time.Now()) {
						valid = true
					}
				}
			}

			if !valid {
				// возвращаем ошибку авторизации 401
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	})
}

// HandlerSignIn POST /signin
// Тело: {"password": "..."}
// При успешной аутентификации устанавливает HttpOnly-куку token с JWT.
func HandlerSignIn(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in interfaces.SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "невалидный JSON")
			return
		}
		secret := cfg.Password
		// Если секрет не задан или пароль пуст — считаем попытку неуспешной
		if secret == "" || strings.TrimSpace(in.Password) == "" {
			writeError(w, http.StatusUnauthorized, "Неверный пароль")
			return
		}
		if !subtleConstantTimeCompare(in.Password, secret) {
			writeError(w, http.StatusUnauthorized, "Неверный пароль")
			return
		}
		// Генерируем JWT с истечением через 8 часов
		expiresAt := time.Now().Add(8 * time.Hour)
		claims := jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		jwtStr, err := token.SignedString([]byte(secret))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка генерации токена")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    jwtStr,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
			Expires:  expiresAt,
		})
		writeJSON(w, http.StatusOK, interfaces.SignInResponse{Token: jwtStr})
	}
}

// сравнение без утечек времени
func subtleConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var res byte = 0
	for i := 0; i < len(a); i++ {
		res |= a[i] ^ b[i]
	}
	return res == 0
}

// HandlerTasks GET /api/tasks
func HandlerTasks(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		list, err := taskService.GetTasks(search, db.DefaultTaskLimit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Преобразуем к TaskResponse для совместимости с тестами
		tasks := make([]interfaces.TaskResponse, 0, len(list))
		for _, t := range list {
			tasks = append(tasks, interfaces.TaskResponse{
				ID:      strconv.FormatInt(t.ID, 10),
				Date:    t.Date,
				Title:   t.Title,
				Comment: t.Comment,
				Repeat:  t.Repeat,
			})
		}
		writeJSON(w, http.StatusOK, interfaces.TasksResponse{Tasks: tasks})
	}
}

// HandlerGetTask GET /api/task?id=...
func HandlerGetTask(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if strings.TrimSpace(idStr) == "" {
			writeError(w, http.StatusBadRequest, "id: обязателен")
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id: неверный формат")
			return
		}
		t, err := taskService.GetTask(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, interfaces.TaskResponse{
			ID:      strconv.FormatInt(t.ID, 10),
			Date:    t.Date,
			Title:   t.Title,
			Comment: t.Comment,
			Repeat:  t.Repeat,
		})
	}
}

// HandlerAddTask POST /api/task
func HandlerAddTask(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in interfaces.AddTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "невалидный JSON")
			return
		}
		id, err := taskService.AddTask(time.Now(),
			strings.TrimSpace(in.Date),
			strings.TrimSpace(in.Title),
			strings.TrimSpace(in.Comment),
			strings.TrimSpace(in.Repeat))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, interfaces.AddTaskResponse{ID: id})
	}
}

// HandlerEditTask PUT /api/task
func HandlerEditTask(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in interfaces.EditTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "невалидный JSON")
			return
		}
		idStr := strings.TrimSpace(in.ID)
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "id: обязателен")
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id: неверный формат")
			return
		}
		err = taskService.UpdateTask(time.Now(), id,
			strings.TrimSpace(in.Date),
			strings.TrimSpace(in.Title),
			strings.TrimSpace(in.Comment),
			strings.TrimSpace(in.Repeat))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, interfaces.EmptyResponse{})
	}
}

// HandlerDeleteTask DELETE /api/task?id=...
func HandlerDeleteTask(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if strings.TrimSpace(idStr) == "" {
			writeError(w, http.StatusBadRequest, "id: обязателен")
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id: неверный формат")
			return
		}
		if err := taskService.DeleteTask(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, interfaces.EmptyResponse{})
	}
}

// HandlerDone POST /api/task/done?id=...
func HandlerDone(taskService *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if strings.TrimSpace(idStr) == "" {
			writeError(w, http.StatusBadRequest, "id: обязателен")
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id: неверный формат")
			return
		}
		if err := taskService.DoneTask(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, interfaces.EmptyResponse{})
	}
}
