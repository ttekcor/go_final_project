package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"main/pkg/scheduler"
	"main/pkg/service"

	"github.com/golang-jwt/jwt/v5"
)

func HandlerHTML(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Ошибка чтения файла", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// writeJSON — утилита для JSON-ответов
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusOK, map[string]any{"error": msg})
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
		now, err = time.Parse("20060102", nowStr)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("now: некорректная дата"))
			return
		}
	}
	next, err := scheduler.NextDate(now, dstart, repeat)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(next))
}

func Auth(next http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // смотрим наличие пароля
        pass := os.Getenv("TODO_PASSWORD")
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
func HandlerSignIn(w http.ResponseWriter, r *http.Request) {
	type signInRequest struct {
		Password string `json:"password"`
	}
	var in signInRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, "невалидный JSON")
		return
	}
	secret := strings.TrimSpace(os.Getenv("TODO_PASSWORD"))
	// Если секрет не задан или пароль пуст — считаем попытку неуспешной
	if secret == "" || strings.TrimSpace(in.Password) == "" {
		writeError(w, "Неверный пароль")
		return
	}
	if !subtleConstantTimeCompare(in.Password, secret) {
		writeError(w, "Неверный пароль")
		return
	}
	// Генерируем JWT с истечением через 30 дней
	expiresAt := time.Now().Add(8 * time.Hour)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtStr, err := token.SignedString([]byte(secret))
	if err != nil {
		writeError(w, "Неверный пароль")
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
	writeJSON(w, http.StatusOK, map[string]any{"token": jwtStr})
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
func HandlerTasks(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	list, err := service.GetTasks(search, 50)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	// Преобразуем к []map[string]string для совместимости с тестами
	tasks := make([]map[string]string, 0, len(list))
	for _, t := range list {
		tasks = append(tasks, map[string]string{
			"id":      strconv.FormatInt(t.ID, 10),
			"date":    t.Date,
			"title":   t.Title,
			"comment": t.Comment,
			"repeat":  t.Repeat,
		})
	}
	if tasks == nil {
		tasks = []map[string]string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// HandlerGetTask GET /api/task?id=...
func HandlerGetTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if strings.TrimSpace(idStr) == "" {
		writeError(w, "id: обязателен")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "id: неверный формат")
		return
	}
	t, err := service.GetTask(id)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"id":      strconv.FormatInt(t.ID, 10),
		"date":    t.Date,
		"title":   t.Title,
		"comment": t.Comment,
		"repeat":  t.Repeat,
	})
}

// HandlerAddTask POST /api/task
func HandlerAddTask(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, "невалидный JSON")
		return
	}
	id, err := service.AddTask(time.Now(),
		strings.TrimSpace(in.Date),
		strings.TrimSpace(in.Title),
		strings.TrimSpace(in.Comment),
		strings.TrimSpace(in.Repeat))
	if err != nil {
		writeError(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

// HandlerEditTask PUT /api/task
func HandlerEditTask(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, "невалидный JSON")
		return
	}
	idStr := strings.TrimSpace(fmtAny(in["id"]))
	if idStr == "" {
		writeError(w, "id: обязателен")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "id: неверный формат")
		return
	}
	err = service.UpdateTask(time.Now(), id,
		strings.TrimSpace(fmtAny(in["date"])),
		strings.TrimSpace(fmtAny(in["title"])),
		strings.TrimSpace(fmtAny(in["comment"])),
		strings.TrimSpace(fmtAny(in["repeat"])))
	if err != nil {
		writeError(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// HandlerDeleteTask DELETE /api/task?id=...
func HandlerDeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if strings.TrimSpace(idStr) == "" {
		writeError(w, "id: обязателен")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "id: неверный формат")
		return
	}
	if err := service.DeleteTask(id); err != nil {
		writeError(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// HandlerDone POST /api/task/done?id=...
func HandlerDone(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if strings.TrimSpace(idStr) == "" {
		writeError(w, "id: обязателен")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "id: неверный формат")
		return
	}
	if err := service.DoneTask(id); err != nil {
		writeError(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func fmtAny(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(v)
		return string(b)[1:len(string(b))-1]
	}
}
