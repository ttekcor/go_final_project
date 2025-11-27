package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var Port = func() int {
	if v := os.Getenv("TODO_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return 7540
}()

var DBFile = "../scheduler.db"
var FullNextDate = true
var Search = true
var Token = initToken()

// initToken получает JWT токен через /api/signin для использования в тестах
func initToken() string {
	password := os.Getenv("TODO_PASSWORD")
	if password == "" {
		// Если пароль не установлен, авторизация не требуется
		return ""
	}

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/api/signin", Port)
	body := map[string]string{"password": password}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return ""
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return ""
	}

	if token, ok := result["token"].(string); ok {
		return token
	}

	return ""
}
