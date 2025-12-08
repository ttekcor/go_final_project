package interfaces

// Типы ответов

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}

// SignInResponse представляет ответ на авторизацию
type SignInResponse struct {
	Token string `json:"token"`
}

// TaskResponse представляет ответ с одной задачей
type TaskResponse struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// TasksResponse представляет ответ со списком задач
type TasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}

// AddTaskResponse представляет ответ на создание задачи
type AddTaskResponse struct {
	ID int64 `json:"id"`
}

// EmptyResponse представляет пустой ответ
type EmptyResponse struct{}
