FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/oncall-agent ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/oncall-agent /app/oncall-agent
COPY configs /app/configs
RUN mkdir -p /app/data/uploads

EXPOSE 8080
CMD ["/app/oncall-agent"]
