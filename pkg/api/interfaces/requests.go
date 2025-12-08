package interfaces

// Типы запросов

// SignInRequest представляет запрос на авторизацию
type SignInRequest struct {
	Password string `json:"password"`
}

// AddTaskRequest представляет запрос на создание задачи
type AddTaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// EditTaskRequest представляет запрос на изменение задачи
type EditTaskRequest struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}
