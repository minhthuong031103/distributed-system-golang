# service-a/Dockerfile
FROM golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o service-a

EXPOSE 8080

CMD ["./service-a"]
