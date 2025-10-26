# Stack Tecnológica

## Linguagens e Runtime

### Go 1.18+
A biblioteca **go-smpp** é implementada em Go (Golang) e requer **Go 1.18 ou superior**.

**Razões para escolha do Go**:
- Excelente suporte para programação concorrente (goroutines e channels)
- Performance nativa próxima a C/C++
- Biblioteca padrão robusta para networking (TCP/TLS)
- Tipagem estática forte com interfaces flexíveis
- Compilação para binário único sem dependências externas
- Garbage collector eficiente para aplicações de longa duração
- Amplamente adotado para infraestrutura e sistemas distribuídos

**Versão Mínima**: Go 1.18 (necessária para suporte a generics e melhorias na biblioteca padrão)

---

## Frameworks Principais

Esta biblioteca **não utiliza frameworks externos pesados**, aproveitando principalmente a biblioteca padrão do Go. A filosofia é manter a biblioteca leve, com dependências mínimas.

### Biblioteca Padrão Go
- **net**: TCP/TLS connections, dialing, listening
- **io**: Readers/Writers para serialização de PDUs
- **context**: Gerenciamento de timeouts e cancelamento
- **sync**: Mutexes, WaitGroups para sincronização
- **time**: Timers para keepalive, backoff, timeouts
- **encoding/binary**: Serialização big-endian de inteiros

---

## Bibliotecas Chave

### Dependências Diretas

#### 1. **golang.org/x/text** (v0.10.0)
**Propósito**: Transformação de caracteres e encodings de texto

**Uso no projeto**:
- Base para implementação de codecs personalizados (GSM7)
- Interface `transform.Transformer` para conversões de encoding
- Suporte a encodings internacionais (Latin1, UCS2, Cyrillic)

**Localização**: Usado principalmente em [smpp/pdu/pdutext/](../smpp/pdu/pdutext/) e [smpp/encoding/gsm7.go](../smpp/encoding/gsm7.go)

#### 2. **golang.org/x/time** (v0.3.0)
**Propósito**: Rate limiting e controle de fluxo

**Uso no projeto**:
- Interface `rate.Limiter` para controle de throughput
- Integração opcional para limitar envio de mensagens

**Localização**: Suportado via interface `RateLimiter` em [smpp/transmitter.go](../smpp/transmitter.go)

#### 3. **golang.org/x/net** (v0.11.0)
**Propósito**: Extensões de networking

**Uso no projeto**:
- Suporte a `context.Context` em operações de rede
- Utilitários de rede modernos

**Localização**: Usado em conexões TCP/TLS

#### 4. **github.com/urfave/cli** (v1.22.14)
**Propósito**: Framework para aplicações CLI

**Uso no projeto**:
- Parsing de argumentos e flags para CLI tools
- Geração automática de help text
- Subcomandos para `sms` e `smsapid`

**Localização**: Usado em [cmd/sms/](../cmd/sms/) e [cmd/smsapid/](../cmd/smsapid/)

### Dependências Indiretas

#### github.com/cpuguy83/go-md2man/v2 (v2.0.2)
Conversão de Markdown para man pages (dependência do urfave/cli)

#### github.com/russross/blackfriday/v2 (v2.1.0)
Parser Markdown (dependência do urfave/cli)

---

## Banco de Dados

**Nenhum banco de dados é utilizado diretamente pela biblioteca.**

A biblioteca é **stateless** por design - toda a lógica de persistência (logs de mensagens, histórico, filas) deve ser implementada pela aplicação consumidora.

### Considerações de Persistência
Se você precisar persistir dados SMPP, considere:
- **Redis**: Para cache de mensagens longas em merge, estados de conexão
- **PostgreSQL/MySQL**: Para logs de auditoria, histórico de mensagens, delivery receipts
- **MongoDB**: Para documentos JSON de mensagens e metadados
- **Elasticsearch**: Para busca e análise de logs SMPP

---

## Infraestrutura

### Deployment
A biblioteca **go-smpp** é distribuída como **package Go** e importada em outros serviços.

**Modelo de Uso**:
```go
import "github.com/devyx-tech/go-smpp/smpp"
```

### Containers
Para aplicações que usam go-smpp:
```dockerfile
FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o app ./cmd/your-app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app /app
CMD ["/app"]
```

### Networking
- **Protocolo**: TCP sobre porta **2775** (padrão SMPP) ou **2776** (SMPP over TLS)
- **TLS**: Suportado via configuração `TLS` em Transmitter/Receiver
- **Keep-Alive**: EnquireLink PDUs enviados a cada 10 segundos (configurável)

### Observabilidade
A biblioteca **não inclui** telemetria integrada. Aplicações consumidoras devem implementar:
- **Logging**: Via biblioteca de logging (logrus, zap, zerolog)
- **Métricas**: Prometheus, StatsD para tracking de mensagens enviadas/recebidas
- **Tracing**: OpenTelemetry para distributed tracing de fluxos SMS

---

## Ferramentas de Desenvolvimento

### Build e Compilação
```bash
# Build da biblioteca
go build ./...

# Build das CLI tools
go build -o sms ./cmd/sms
go build -o smsapid ./cmd/smsapid
```

### Testes
```bash
# Executar todos os testes
go test ./...

# Testes com cobertura
go test -cover ./...

# Testes com race detector
go test -race ./...
```

### Linting e Formatação
**Ferramentas Recomendadas**:
- **gofmt**: Formatação padrão Go
- **goimports**: Auto-import de packages
- **golangci-lint**: Linter agregado (recomendado)
- **go vet**: Análise estática da stdlib

```bash
# Formatar código
gofmt -w .

# Análise estática
go vet ./...

# Linting completo (se golangci-lint instalado)
golangci-lint run
```

### CI/CD
O repositório inclui configuração **Travis CI** ([.travis.yml](../.travis.yml)):
- Build automático em cada commit
- Execução de testes
- Verificação de compilação multiplataforma

---

## Arquitetura Geral

### Organização em Camadas

```
┌─────────────────────────────────────────────────────┐
│           Aplicação Consumidora (seu código)        │
└─────────────────────────────────────────────────────┘
                         ▲
                         │ import
                         ▼
┌─────────────────────────────────────────────────────┐
│  Camada de Cliente (Transmitter/Receiver/Transceiver)│
│  - Gestão de conexão e reconexão                    │
│  - EnquireLink keepalive                            │
│  - Rate limiting                                    │
└─────────────────────────────────────────────────────┘
                         ▲
                         ▼
┌─────────────────────────────────────────────────────┐
│     Camada de Protocolo (PDU encoding/decoding)     │
│  - Serialização/deserialização de PDUs              │
│  - Field e TLV handling                             │
│  - Text codecs (GSM7, UCS2, Latin1)                 │
└─────────────────────────────────────────────────────┘
                         ▲
                         ▼
┌─────────────────────────────────────────────────────┐
│         Camada de Rede (TCP/TLS connections)        │
│  - io.Reader/io.Writer                              │
│  - net.Conn abstraction                             │
└─────────────────────────────────────────────────────┘
                         ▲
                         ▼
                   SMSC (Server)
```

### Módulos Principais

#### 1. **smpp/** - Cliente e Servidor
- **transmitter.go**: Cliente para envio de SMS
- **receiver.go**: Cliente para recebimento de SMS
- **transceiver.go**: Cliente bidirecional
- **client.go**: Lógica comum de conexão
- **conn.go**: Wrapper de baixo nível sobre TCP
- **server.go**: Servidor SMPP para testes

#### 2. **smpp/pdu/** - Protocol Data Units
- **pdu.go**: Tipos base (Header, Body, Factory)
- **bind*.go**: PDUs de bind (Transmitter, Receiver, Transceiver)
- **submit*.go**: PDUs de submissão de mensagens
- **deliver*.go**: PDUs de entrega de mensagens
- **query*.go**: PDUs de consulta de status
- **enquirelink.go**: Keepalive PDUs

#### 3. **smpp/pdu/pdufield/** - Campos de PDU
- **field.go**: Tipos de campos (Fixed, Variable, CString, List, Map)
- **defs.go**: Definições de 50+ campos SMPP padrão

#### 4. **smpp/pdu/pdutext/** - Codecs de Texto
- **codec.go**: Interface Codec e implementações
- **gsm7.go**: GSM 7-bit encoding
- **latin1.go**: ISO-8859-1 encoding
- **ucs2.go**: Unicode (UTF-16 big-endian)
- **iso88595.go**: Cyrillic encoding
- **raw.go**: Passthrough sem encoding

#### 5. **smpp/pdu/pdutlv/** - Tag-Length-Value
- **tlv.go**: Estrutura TLV para campos opcionais
- **defs.go**: 45+ tags TLV predefinidas

#### 6. **smpp/encoding/** - Character Encoding
- **gsm7.go**: Implementação GSM 03.38 com escape sequences

#### 7. **smpp/smpptest/** - Test Server
- **smpptest.go**: Servidor SMPP in-process para testes unitários

#### 8. **cmd/** - CLI Tools
- **cmd/sms/**: Cliente CLI para envio/query de SMS
- **cmd/smsapid/**: Daemon API para exposição HTTP

---

## Decisões Arquiteturais Importantes

### 1. Interface-Based Design
**Decisão**: Uso extensivo de interfaces (Conn, Body, Codec, RateLimiter, etc.)

**Razão**:
- Permite substituição de implementações (mocking em testes)
- Facilita extensão sem modificar código core
- Desacopla camadas de abstração

**Trade-off**: Pequeno overhead de indireção, mas ganho enorme em testabilidade

### 2. Channel-Based Status Notifications
**Decisão**: `Bind()` retorna `<-chan ConnStatus` em vez de blocking call

**Razão**:
- Permite tratamento assíncrono de mudanças de estado
- Aplicação pode escolher se bloqueia ou não (`<-conn` vs `select`)
- Goroutines podem monitorar reconexões sem polling

**Trade-off**: Complexidade ligeiramente maior, mas muito mais flexível

### 3. Automatic Reconnection
**Decisão**: Reconexão automática com exponential backoff (max 120s)

**Razão**:
- Conexões SMPP frequentemente caem por timeout ou problemas de rede
- Aplicações de produção precisam ser resilientes
- Reduz código boilerplate nas aplicações consumidoras

**Trade-off**: Aplicação deve monitorar status de conexão para pausar envios se necessário

### 4. In-Memory Message Merging (Receiver)
**Decisão**: Merge de mensagens longas em memória com cleanup periódico

**Razão**:
- Performance: sem I/O para mensagens parciais
- Simplicidade: não requer dependências externas
- SMPP normalmente entrega partes rapidamente (segundos)

**Trade-off**: Mensagens incompletas são perdidas em crash (aceitável para SMS)

### 5. No Built-in Persistence
**Decisão**: Biblioteca não persiste dados (mensagens, logs, estado)

**Razão**:
- Flexibilidade: cada aplicação tem requisitos diferentes
- Simplicidade: mantém biblioteca focada em protocolo SMPP
- Zero dependencies: não impõe escolha de banco de dados

**Trade-off**: Aplicações consumidoras devem implementar persistência se necessário

### 6. Sequence Number Management via Factory
**Decisão**: Interface `pdu.Factory` centraliza geração de sequence numbers

**Razão**:
- Evita estado global (package-level variables)
- Permite múltiplas conexões SMPP com sequences independentes
- Testabilidade: factories mockáveis

**Trade-off**: Aplicações devem criar Factory, mas ganha-se thread-safety

### 7. Separate Transmitter/Receiver/Transceiver
**Decisão**: Três structs distintas em vez de uma única com modos

**Razão**:
- Clareza: type system garante uso correto
- SMPP permite conexões separadas (alguns SMSCs exigem)
- API mais simples: cada tipo expõe apenas métodos relevantes

**Trade-off**: Código duplicado, mas API muito mais clara

### 8. Text Encoding via Codec Interface
**Decisão**: Abstrair encodings (GSM7, UCS2, etc.) via interface comum

**Razão**:
- Extensibilidade: novos encodings adicionáveis sem breaking changes
- Separação de concerns: lógica de encoding isolada
- Reuso: codecs usáveis em qualquer PDU

**Trade-off**: Indireção de interface, mas ganho em manutenibilidade

---

## Dependências do Sistema

### Runtime
- **Sistema Operacional**: Linux, macOS, Windows (cross-platform)
- **Networking**: Acesso TCP à porta do SMSC (geralmente 2775/2776)
- **TLS**: Certificados CA para validação (se usar TLS)

### Desenvolvimento
- **Go Toolchain**: 1.18+
- **Git**: Para clone e versionamento
- **Make** (opcional): Para automação de builds

---

## Performance e Escalabilidade

### Características de Performance
- **Throughput**: Limitado principalmente pelo SMSC e rate limiting
- **Latency**: Tipicamente <100ms para Submit/Response (network-bound)
- **Memory**: Footprint baixo (~10-50MB por conexão)
- **CPU**: Mínimo (network I/O é gargalo, não CPU)

### Escalabilidade
- **Conexões Múltiplas**: Aplicação pode abrir múltiplos Transmitters/Receivers
- **Goroutines**: Cada conexão roda em goroutines independentes
- **Horizontal Scaling**: Deploy múltiplas instâncias com load balancer

### Limitações
- **Window Size**: Configurável (padrão sem limite), SMSCs geralmente limitam a 10-100
- **Rate Limiting**: Opcional, mas recomendado para respeitar limites do SMSC
- **Long Message Merging**: Limitado por memória (cleanup após `MergeCleanupInterval`)

---

## Segurança

### Autenticação
- **SystemID + Password**: Enviados em plaintext no PDU de Bind
- **Recomendação**: Sempre usar TLS para conexões SMPP em produção

### TLS/SSL
- **Suporte**: Via campo `TLS` em Transmitter/Receiver (tipo `*tls.Config`)
- **Validação de Certificados**: Configurável via `tls.Config`

### Secrets Management
- **Não incluído**: Aplicação deve gerenciar credenciais SMPP
- **Recomendação**: Usar variáveis de ambiente ou secret managers (Vault, AWS Secrets Manager)

---

## Próximos Passos

Consulte os outros documentos para entender:
- [**Padrões de Design**](patterns.md) - Como o código está organizado
- [**Funcionalidades**](features.md) - Detalhes de cada feature
- [**Regras de Negócio**](business-rules.md) - Protocolo SMPP implementado
- [**Integração**](integrations.md) - Como usar em seus projetos
