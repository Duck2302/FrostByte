FROM golang:1.24.1

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .


RUN CGO_ENABLED=1 go build main.go

EXPOSE 8080

CMD ["./main"]