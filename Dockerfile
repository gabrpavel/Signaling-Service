FROM golang:1.22.3 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o signaling ./cmd/sso/main.go

FROM golang:1.22.3

WORKDIR /app

COPY --from=builder /app/signaling /app/signaling

COPY ./config ./config

ENTRYPOINT ["/app/signaling"]
CMD ["--config=./config/prod.yaml"]
