FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o chasqui-local-agent .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/chasqui-local-agent .
EXPOSE 5050
ENTRYPOINT ["/app/chasqui-local-agent"]
