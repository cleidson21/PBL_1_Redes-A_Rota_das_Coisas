# Estágio de compilação
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o atuador main.go

# Estágio final
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/atuador .

# Expõe a porta TCP do atuador
EXPOSE 8081/tcp

CMD ["./atuador"]