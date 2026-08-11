FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/ \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker/

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder /server /server
COPY --from=builder /worker /worker
EXPOSE 8080
CMD ["/server"]
