# zpigo

API REST para gerenciamento de sessões do WhatsApp usando Go e WhatsApp Web Multi-Device API.

## Características

- ✅ Gerenciamento completo de sessões do WhatsApp
- ✅ Suporte a múltiplas sessões simultâneas
- ✅ Autenticação via QR Code ou emparelhamento por telefone
- ✅ Configuração de proxy por sessão
- ✅ API REST com endpoints bem definidos
- ✅ Banco de dados PostgreSQL com SQL nativo
- ✅ Arquitetura limpa e modular

## Tecnologias

- **Go 1.23+** - Linguagem de programação
- **Chi Router** - Router HTTP minimalista
- **PostgreSQL** - Banco de dados
- **SQL Nativo** - Queries SQL diretas com database/sql
- **WhatsApp Web Multi-Device** - Biblioteca whatsmeow

## Instalação

1. Clone o repositório:
```bash
git clone <repository-url>
cd zpigo
```

2. Instale as dependências:
```bash
go mod download
```

3. Configure o banco de dados PostgreSQL e copie o arquivo de configuração:
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

4. Execute as migrações do banco:
```bash
go run cmd/server/main.go
```

## Configuração

Edite o arquivo `.env` com suas configurações:

```env
# Servidor
SERVER_PORT=8080
SERVER_HOST=localhost

# Banco de dados
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=zpigo
DB_SSLMODE=disable

# Aplicação
APP_ENV=development
DEBUG=true
```

## Uso

### Iniciar o servidor

```bash
go run cmd/server/main.go
```

O servidor estará disponível em `http://localhost:8080`

### Endpoints da API

#### Sessões

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/sessions/add` | Cria uma nova sessão |
| GET | `/api/v1/sessions/list` | Lista todas as sessões |
| GET | `/api/v1/sessions/{sessionID}/info` | Informações da sessão |
| DELETE | `/api/v1/sessions/{sessionID}` | Remove uma sessão |
| POST | `/api/v1/sessions/{sessionID}/connect` | Conecta a sessão |
| POST | `/api/v1/sessions/{sessionID}/logout` | Faz logout da sessão |
| GET | `/api/v1/sessions/{sessionID}/qr` | Gera QR Code |
| POST | `/api/v1/sessions/{sessionID}/pairphone` | Emparelha telefone |
| POST | `/api/v1/sessions/{sessionID}/proxy/set` | Configura proxy |

### Exemplos de uso

#### Criar uma sessão
```bash
curl -X POST http://localhost:8080/api/v1/sessions/add \
  -H "Content-Type: application/json" \
  -d '{"name": "Minha Sessão"}'
```

#### Listar sessões
```bash
curl http://localhost:8080/api/v1/sessions/list
```

#### Obter QR Code
```bash
curl http://localhost:8080/api/v1/sessions/{sessionID}/qr
```

#### Conectar sessão
```bash
curl -X POST http://localhost:8080/api/v1/sessions/{sessionID}/connect
```

#### Emparelhar telefone
```bash
curl -X POST http://localhost:8080/api/v1/sessions/{sessionID}/pairphone \
  -H "Content-Type: application/json" \
  -d '{"phoneNumber": "+5511999999999", "code": "123456"}'
```

#### Configurar proxy
```bash
curl -X POST http://localhost:8080/api/v1/sessions/{sessionID}/proxy/set \
  -H "Content-Type: application/json" \
  -d '{
    "host": "proxy.example.com",
    "port": 8080,
    "type": "http",
    "username": "user",
    "password": "pass"
  }'
```

## Estrutura do Projeto

```
zpigo/
├── cmd/
│   └── server/
│       └── main.go              # Ponto de entrada da aplicação
├── internal/
│   ├── api/
│   │   ├── dto/                 # Data Transfer Objects
│   │   ├── handlers/            # Handlers HTTP
│   │   └── router/              # Configuração de rotas
│   ├── app/                     # Configuração da aplicação
│   ├── config/                  # Configurações
│   ├── store/                   # Store unificado
│   │   ├── models/              # Modelos de dados
│   │   └── migrations/          # Migrações SQL
│   ├── meow/                    # Gerenciador WhatsApp
│   └── repository/              # Camada de dados
├── .env.example                 # Exemplo de configuração
├── go.mod                       # Dependências Go
└── README.md                    # Documentação
```

## Desenvolvimento

### Executar em modo desenvolvimento
```bash
go run cmd/server/main.go
```

### Compilar para produção
```bash
go build -o bin/server cmd/server/main.go
```

### Executar testes
```bash
go test ./...
```

## Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## Licença

Este projeto está sob a licença MIT. Veja o arquivo `LICENSE` para mais detalhes.