## Планировщик задач (Todo Scheduler)

Небольшой веб‑сервер с фронтендом для планирования задач. Приложение хранит данные в SQLite (`scheduler.db`), отдает веб‑интерфейс из каталога `web` и предоставляет HTTP API. Часть операций (создание/изменение/удаление задач, отметка выполнения) защищены авторизацией по паролю через JWT.

- Бэкенд: Go + `net/http`, `github.com/go-chi/chi/v5` (роутинг)
- Хранение: SQLite (`modernc.org/sqlite`), файл БД в рабочей директории
- UI: статические файлы из каталога `web`

### Задания со звёздочкой

Выполнена Аутенцификация и работа с Docker

## Локальный запуск

Требуется установленный Go.

1. Опционально создайте файл `.env` в корне проекта (переменные читаются автоматически):

```
TODO_PORT=7541
TODO_PASSWORD=changeme
```

2. Запустите сервер из корня репозитория:

```bash
go run .
```

3. Откройте в браузере адрес (если `TODO_PORT` не задан — по умолчанию 7541):

```
http://localhost:7541/
```

Примечания:

- Файл БД `scheduler.db` будет создан автоматически при первом запуске (если его нет).
- Для входа в UI используйте пароль из `TODO_PASSWORD`. Если переменная не задана, аутентификация не пройдет.

## Запуск тестов

Запуск из корня проекта:

```bash
go test ./tests -v
```

Параметры в `tests/settings.go`:

- `Port`: берется из переменной окружения `TODO_PORT`, если не задана — `7541`.
- `DBFile`: `../scheduler.db` (файл БД относительно каталога `tests`).
- `FullNextDate`, `Search`: включены.
- `Token`: пустой (для незащищенных запросов).

Примеры установки переменных окружения перед тестами:

```powershell
# Windows PowerShell
$env:TODO_PORT="7541"; go test ./tests -v
```

```bash
# Linux/macOS
TODO_PORT=7541 go test ./tests -v
```

## Docker: сборка и запуск

В проекте есть `Dockerfile` (multi-stage). На этапе рантайма используется `ubuntu:latest`. По умолчанию в образе установлено `ENV TODO_PORT=7541`.

1. Сборка образа:

```bash
docker build -t go-final-todo:latest .
```

2. Создайте файл БД на хосте (нужен для корректного bind‑mount):

```powershell
# Windows PowerShell
if (!(Test-Path .\scheduler.db)) { New-Item -Path .\scheduler.db -ItemType File | Out-Null }
```

```bash
# Linux/macOS
test -f ./scheduler.db || : > ./scheduler.db
```

3. Запуск контейнера (порт и пароль можно менять):

```powershell
docker run --rm -it `
  -p 7541:7541 `
  --name go-final-todo `
  --env TODO_PASSWORD="mySecret" `
  --mount type=bind,source="${PWD}\scheduler.db",target=/app/scheduler.db `
  go-final-todo:latest
```

```bash
# Linux/macOS
docker run --rm -it \
  -p 7541:7541 \
  --name go-final-todo \
  -e TODO_PASSWORD="mySecret" \
  -v "$(pwd)/scheduler.db:/app/scheduler.db" \
  go-final-todo:latest
```

Откройте в браузере:

```
http://localhost:7541/
```

Заметки:

- Можно переопределить порт запуском с `-e TODO_PORT=8080` и пробросом `-p 8080:8080`.
- Если видите, что сервер слушает `:7540`, убедитесь, что в контейнер передан `TODO_PORT=7541` или сопоставьте порты `-p 7541:7540`.
- Для входа в UI используйте пароль из `TODO_PASSWORD`.
