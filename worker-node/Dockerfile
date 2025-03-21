FROM golang:1.24.1

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o main main.go database_handler.go

RUN go get go.mongodb.org/mongo-driver/mongo

EXPOSE 8081

CMD ["./main"]