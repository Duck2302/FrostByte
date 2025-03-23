FROM golang:1.24.1

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN go get go.mongodb.org/mongo-driver/mongo

RUN CGO_ENABLED=1 go build main.go worker_handler.go file_handler.go chunk_handler.go database_handler.go


EXPOSE 8080

CMD ["./main"]