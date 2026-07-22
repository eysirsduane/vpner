FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/just-vpn-origin .

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /app/upload

COPY --from=builder /out/just-vpn-origin /app/just-vpn-origin
COPY conf /app/conf

ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["/app/just-vpn-origin"]
