FROM golang:1.15-alpine

COPY . /app

WORKDIR /app

RUN go build ./...

CMD ["./pinger"]