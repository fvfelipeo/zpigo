# Build stage
FROM golang:1.23-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates tzdata

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar a aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Production stage
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Criar usuário não-root
RUN addgroup -g 1001 -S zpigo && \
    adduser -u 1001 -S zpigo -G zpigo

# Definir diretório de trabalho
WORKDIR /app

# Copiar binário da aplicação
COPY --from=builder /app/main .

# Copiar arquivos de configuração se necessário
COPY --from=builder /app/.env.example .env.example

# Alterar proprietário dos arquivos
RUN chown -R zpigo:zpigo /app

# Usar usuário não-root
USER zpigo

# Expor porta
EXPOSE 8080

# Comando para executar a aplicação
CMD ["./main"]
