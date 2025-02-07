FROM golang:1.23.5-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

CMD ["go", "test", "-v", "./..."]