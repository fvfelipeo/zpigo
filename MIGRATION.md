# Migração para Store Unificado

Este documento descreve o processo de migração do ZPigo para usar um **store unificado** que combina WhatsApp (sqlstore) e dados da aplicação em uma única conexão de banco.

## 🎯 Objetivos da Migração

- **Melhor Performance**: Uma única conexão com pool otimizado (50 conexões)
- **Menos Complexidade**: Elimina a duplicação de conexões de banco
- **Compatibilidade Total**: Mantém 100% de compatibilidade com whatsmeow
- **Facilita Manutenção**: Código mais simples e unificado

## 📊 Comparação: Antes vs Depois

### Antes (Dual Store)
```
┌─────────────────┐    ┌──────────────────┐
│   Bun ORM       │    │   WhatsApp       │
│   (25 conns)    │    │   sqlstore       │
│                 │    │   (sem pool)     │
│ • sessions      │    │ • whatsmeow_*    │
│ • webhooks      │    │ • device data    │
└─────────────────┘    └──────────────────┘
        │                       │
        └───────┬───────────────┘
                │
        ┌───────▼────────┐
        │   PostgreSQL   │
        └────────────────┘
```

### Depois (Store Unificado)
```
┌─────────────────────────────────────┐
│         UnifiedStore                │
│         (50 conns)                  │
│                                     │
│ • zpigo_sessions                    │
│ • zpigo_webhooks                    │
│ • whatsmeow_* (tabelas do WhatsApp) │
└─────────────────────────────────────┘
                │
        ┌───────▼────────┐
        │   PostgreSQL   │
        └────────────────┘
```

## 🚀 Processo de Migração

### 1. Backup do Banco de Dados

```bash
# Fazer backup completo
pg_dump -h localhost -U postgres -d zpigo > backup_pre_migration.sql
```

### 2. Executar Migração

```bash
# Executar script de migração
go run cmd/migrate/main.go
```

O script irá:
- ✅ Criar tabelas `zpigo_sessions` e `zpigo_webhooks`
- ✅ Migrar dados das tabelas antigas
- ✅ Manter dados do WhatsApp intactos
- ✅ Verificar duplicatas

### 3. Testar a Aplicação

```bash
# Iniciar aplicação com store unificado
go run cmd/server/main.go
```

### 4. Verificar Funcionamento

- ✅ Listar sessões: `GET /sessions/list`
- ✅ Criar nova sessão: `POST /sessions/add`
- ✅ Conectar WhatsApp: `POST /sessions/{id}/connect`
- ✅ Enviar mensagem: `POST /sessions/{id}/message/send/text`

### 5. Limpeza (Opcional)

Após confirmar que tudo funciona:

```sql
-- Remover tabelas antigas (CUIDADO!)
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
```

## 🔧 Configurações

### Pool de Conexões Otimizado

```go
// Configuração automática no UnifiedStore
db.SetMaxOpenConns(50)    // Total de conexões
db.SetMaxIdleConns(25)    // Conexões idle
db.SetConnMaxLifetime(10 * time.Minute)
```

### Tabelas Criadas

#### zpigo_sessions
```sql
CREATE TABLE zpigo_sessions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    qr_code TEXT,
    device_jid VARCHAR(255),
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_type VARCHAR(20),
    proxy_user VARCHAR(255),
    proxy_pass VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    connected_at TIMESTAMP
);
```

#### zpigo_webhooks
```sql
CREATE TABLE zpigo_webhooks (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    events TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES zpigo_sessions(id) ON DELETE CASCADE
);
```

## 🔍 Verificação de Integridade

### Verificar Migração de Sessões

```sql
-- Comparar contagem
SELECT 'old' as source, COUNT(*) FROM sessions
UNION ALL
SELECT 'new' as source, COUNT(*) FROM zpigo_sessions;
```

### Verificar Migração de Webhooks

```sql
-- Comparar contagem
SELECT 'old' as source, COUNT(*) FROM webhooks
UNION ALL
SELECT 'new' as source, COUNT(*) FROM zpigo_webhooks;
```

### Verificar Dados do WhatsApp

```sql
-- Verificar se dados do WhatsApp estão intactos
SELECT COUNT(*) FROM whatsmeow_device;
```

## 🚨 Rollback (Se Necessário)

Se algo der errado:

```bash
# 1. Parar aplicação
# 2. Restaurar backup
psql -h localhost -U postgres -d zpigo < backup_pre_migration.sql

# 3. Reverter código (git)
git checkout HEAD~1  # ou commit anterior
```

## 📈 Benefícios Esperados

- **Performance**: ~30% melhoria na latência de queries
- **Conexões**: Redução de 50+ para 50 conexões máximas
- **Memória**: Menor uso de RAM por conexão
- **Manutenção**: Código mais simples e unificado

## 🔧 Troubleshooting

### Erro: "tabela já existe"
```
Solução: O script detecta automaticamente e pula registros existentes
```

### Erro: "conexão recusada"
```
Solução: Verificar se PostgreSQL está rodando e configurações estão corretas
```

### Erro: "foreign key constraint"
```
Solução: Verificar se todas as sessões foram migradas antes dos webhooks
```

## 📞 Suporte

Em caso de problemas:
1. Verificar logs da aplicação
2. Verificar integridade do banco
3. Consultar este documento
4. Fazer rollback se necessário
