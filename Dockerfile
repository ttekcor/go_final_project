## Этап сборки (Linux бинарник)
FROM golang:latest AS builder
WORKDIR /src

# Заранее подтянуть зависимости для кэширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект и собираем статический бинарник под Linux
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /out/app .

## Рантайм-этап
FROM ubuntu:latest
WORKDIR /app

# Копируем только бинарник и статику
COPY --from=builder /out/app /app/app
COPY web/ /app/web/

# Значения по умолчанию (можно переопределить при запуске контейнера)
ENV TODO_PORT=7541

# Порт HTTP-сервера
EXPOSE 7541

# По умолчанию сервер ищет SQLite-файл scheduler.db в /app
# Рекомендуется примонтировать файл БД с хоста в /app/scheduler.db
ENTRYPOINT ["./app"]


