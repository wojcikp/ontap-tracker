FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o app ./cmd/app/main.go

RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o app ./cmd/app/main.go

FROM alpine:3.19

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 3000

CMD ["./app"]
