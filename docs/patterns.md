# Padrões de Design

Este documento descreve os padrões arquiteturais, de código e convenções utilizados na biblioteca **go-smpp**.

---

## Padrões Arquiteturais

### 1. Layered Architecture (Arquitetura em Camadas)

A biblioteca está organizada em **4 camadas distintas**:

```
┌────────────────────────────────────────┐
│  Application Layer (consumidor)       │
├────────────────────────────────────────┤
│  Client Layer (Tx/Rx/Transceiver)     │  ← smpp/*.go
├────────────────────────────────────────┤
│  Protocol Layer (PDU handling)        │  ← smpp/pdu/*.go
├────────────────────────────────────────┤
│  Transport Layer (TCP/TLS)            │  ← net.Conn
└────────────────────────────────────────┘
```

**Responsabilidades**:
- **Application**: Lógica de negócio, persistência, API
- **Client**: Gestão de conexão, keepalive, reconexão, dispatching
- **Protocol**: Encoding/decoding de PDUs, validação de campos
- **Transport**: I/O de rede, TLS, timeouts

**Benefícios**:
- Separação clara de concerns
- Testabilidade: cada camada testável independentemente
- Substituição: camadas podem ser mockadas/substituídas

---

### 2. Clean Architecture Principles

A biblioteca segue princípios de Clean Architecture:

**Dependency Rule**: Dependências apontam para dentro (Application → Client → Protocol → Transport)

**Interfaces no Core**: [smpp/client.go:27-32](../smpp/client.go#L27-L32)
```go
type Conn interface {
    Read() (pdu.Body, error)
    Write(pdu.Body) error
    Close() error
}
```

**Implementations nas Bordas**: Implementações concretas (TCP, mock) ficam em pacotes específicos.

---

### 3. Event-Driven Communication

Uso de **channels** para comunicação assíncrona entre goroutines:

**Status Updates**: [smpp/transmitter.go:73](../smpp/transmitter.go#L73)
```go
func (t *Transmitter) Bind() <-chan ConnStatus {
    // Retorna channel para notificações de status
}
```

**Request/Response Correlation**: [smpp/transmitter.go:189-195](../smpp/transmitter.go#L189-L195)
```go
// inflight map para correlacionar requests com responses
inflight := make(map[uint32]chan *tx)
```

**Handler-based Processing**: [smpp/receiver.go:45](../smpp/receiver.go#L45)
```go
type HandlerFunc func(pdu.Body)
```

**Benefícios**:
- Non-blocking: aplicação não trava em I/O
- Concorrência: múltiplas operações paralelas
- Desacoplamento: produtor e consumidor independentes

---

## Padrões de Código

### 1. Factory Pattern

**Uso**: Criação de PDUs com sequence number management

**Interface**: [smpp/pdu/pdu.go:21-23](../smpp/pdu/pdu.go#L21-L23)
```go
type Factory interface {
    NewSeq() (seq uint32)
}
```

**Implementação**: [smpp/client.go:78-82](../smpp/client.go#L78-L82)
```go
type factory struct{ seq uint32 }

func (f *factory) NewSeq() uint32 {
    return atomic.AddUint32(&f.seq, 1)
}
```

**Benefícios**:
- Evita estado global (package-level variables)
- Thread-safe: usa atomic operations
- Testável: factories podem ser mockadas

---

### 2. Strategy Pattern

**Uso**: Múltiplas estratégias de encoding de texto

**Interface**: [smpp/pdu/pdutext/codec.go:15-20](../smpp/pdu/pdutext/codec.go#L15-L20)
```go
type Codec interface {
    Type() DataCoding
    Encode() []byte
    Decode() []byte
}
```

**Implementações**:
- `GSM7`: [pdutext/gsm7.go](../smpp/pdu/pdutext/gsm7.go)
- `Latin1`: [pdutext/latin1.go](../smpp/pdu/pdutext/latin1.go)
- `UCS2`: [pdutext/ucs2.go](../smpp/pdu/pdutext/ucs2.go)
- `Raw`: [pdutext/raw.go](../smpp/pdu/pdutext/raw.go)

**Uso**:
```go
sm := &smpp.ShortMessage{
    Text: pdutext.GSM7("Olá"),  // Estratégia GSM7
}
```

**Benefícios**:
- Extensível: novos codecs sem alterar código existente
- Intercambiável: aplicação escolhe encoding em runtime
- Isolado: lógica de encoding encapsulada

---

### 3. Adapter Pattern

**Uso**: Wrapping de conexões com middleware

**Interface**: [smpp/client.go:34-36](../smpp/client.go#L34-L36)
```go
type ConnMiddleware func(Conn) Conn
```

**Implementação**: [smpp/client.go:194-215](../smpp/client.go#L194-L215)
```go
type connSwitch struct {
    mu   sync.RWMutex
    conn Conn
}

func (c *connSwitch) Read() (pdu.Body, error) {
    c.mu.RLock()
    conn := c.conn
    c.mu.RUnlock()
    return conn.Read()
}
```

**Benefícios**:
- Composição: múltiplos middlewares encadeados
- Transparência: aplicação não sabe que está wrapped
- Extensibilidade: logging, metrics, tracing via middleware

---

### 4. Observer Pattern

**Uso**: Notificação de mudanças de estado de conexão

**Implementação**: [smpp/client.go:53-67](../smpp/client.go#L53-L67)
```go
type ConnStatus struct {
    status Status
    err    error
}

func (c ConnStatus) Status() Status { return c.status }
func (c ConnStatus) Error() error   { return c.err }
```

**Uso na Aplicação**:
```go
conn := tx.Bind()
go func() {
    for status := range conn {
        if status.Status() == Connected {
            log.Println("Conectado!")
        }
    }
}()
```

**Benefícios**:
- Reativo: aplicação responde a mudanças
- Desacoplado: cliente não precisa polling
- Assíncrono: não bloqueia execução

---

### 5. Template Method Pattern

**Uso**: Estrutura comum de PDUs com variações específicas

**Base Abstrata**: [smpp/pdu/pdu.go:32-50](../smpp/pdu/pdu.go#L32-L50)
```go
type Codec struct {
    H   *Header
    L   pdufield.List
    F   pdufield.Map
    TLV pdutlv.Map
}

func (c *Codec) Header() *Header { return c.H }
func (c *Codec) Fields() pdufield.Map { return c.F }
```

**Implementações Especializadas**: [smpp/pdu/submit_sm.go](../smpp/pdu/submit_sm.go)
```go
type SubmitSM struct {
    *Codec
}

func (p *SubmitSM) FieldList() pdufield.List {
    return pdufield.List{
        pdufield.ServiceType,
        pdufield.SourceAddrTON,
        // ... campos específicos de SubmitSM
    }
}
```

**Benefícios**:
- Reuso: lógica comum em base class
- Variação: cada PDU define seus campos específicos
- Consistência: todos PDUs seguem mesma estrutura

---

### 6. Connection Pool Pattern (Implicit)

**Uso**: Gerenciamento de conexões persistentes com reconexão

**Implementação**: [smpp/transmitter.go:137-175](../smpp/transmitter.go#L137-L175)
```go
func (t *Transmitter) Bind() <-chan ConnStatus {
    // Loop infinito de reconexão
    for {
        conn, err := dial(t.Addr, t.TLS)
        if err != nil {
            // Exponential backoff
            time.Sleep(backoff)
            continue
        }
        // Usa conexão até falhar
    }
}
```

**Características**:
- **Persistent**: conexão mantida viva por EnquireLink
- **Resilient**: reconexão automática em falhas
- **Backoff**: exponential backoff (1s → 2s → 4s → ... → max 120s)

**Benefícios**:
- Disponibilidade: aplicação sempre tenta reconectar
- Performance: evita overhead de criar conexão a cada mensagem
- Transparência: aplicação não precisa gerenciar reconexões

---

### 7. Request-Response Correlation Pattern

**Uso**: Correlacionar requisições SMPP com respostas via sequence numbers

**Implementação**: [smpp/transmitter.go:189-236](../smpp/transmitter.go#L189-L236)
```go
inflight := make(map[uint32]chan *tx)

// Enviar request
seq := p.Header().Seq
rc := make(chan *tx, 1)
inflight[seq] = rc

conn.Write(p)

// Aguardar response
select {
case resp := <-rc:
    return resp, nil
case <-time.After(t.RespTimeout):
    return nil, ErrTimeout
}
```

**Benefícios**:
- Concorrência: múltiplas requests paralelas
- Timeout: detecta respostas perdidas
- Ordenação: responses podem chegar fora de ordem

---

## Organização de Código

### Estrutura de Pacotes

```
smpp/
├── *.go                      # Tipos principais (Transmitter, Receiver, etc.)
├── pdu/
│   ├── *.go                  # Tipos de PDU (SubmitSM, DeliverSM, etc.)
│   ├── pdufield/
│   │   ├── field.go          # Tipos de campos
│   │   └── defs.go           # Definições de campos SMPP
│   ├── pdutext/
│   │   ├── codec.go          # Interface Codec
│   │   ├── gsm7.go           # Codec GSM7
│   │   ├── latin1.go         # Codec Latin1
│   │   ├── ucs2.go           # Codec UCS2
│   │   └── ...
│   └── pdutlv/
│       ├── tlv.go            # Estrutura TLV
│       └── defs.go           # Definições de tags TLV
├── encoding/
│   └── gsm7.go               # Encoder/Decoder GSM 03.38
├── smpptest/
│   └── smpptest.go           # Test server
└── ...
```

### Princípios de Organização

**1. Pacotes por Feature**: Cada subpacote representa uma funcionalidade clara
- `pdu/`: Protocol Data Units
- `pdu/pdufield/`: Campos de PDU
- `pdu/pdutext/`: Text encoding
- `pdu/pdutlv/`: Tag-Length-Value fields
- `encoding/`: Character encodings
- `smpptest/`: Test utilities

**2. Flat Hierarchy**: Evita hierarquias profundas
- Máximo 3 níveis: `smpp/pdu/pdutext/`
- Facilita navegação e imports

**3. Separation of Concerns**: Cada pacote tem responsabilidade única
- `pdufield`: Apenas tipos e parsing de campos
- `pdutext`: Apenas codecs de texto
- `encoding`: Apenas transformações de caracteres

---

## Convenções de Nomenclatura

### Pacotes
- **Lowercase**: `smpp`, `pdu`, `pdufield`, `pdutext`, `pdutlv`
- **Singular**: `encoding` (não `encodings`)
- **Descritivo**: `smpptest` (não `test` - conflitaria com stdlib)

### Tipos
- **PascalCase**: `Transmitter`, `SubmitSM`, `ConnStatus`
- **Exported**: Tipos públicos iniciam com maiúscula
- **Descritivo**: `BindTransmitter` (não `BT`)

### Interfaces
- **Noun**: `Conn`, `Body`, `Codec`, `Factory` (não `Connectable`, `Codable`)
- **Exceção**: `-er` para interfaces de comportamento (`Reader`, `Writer`, `Closer`)

### Funções e Métodos
- **PascalCase (exported)**: `Bind()`, `Submit()`, `NewSeq()`
- **camelCase (internal)**: `dial()`, `bindFunc()`, `submitMsg()`
- **Getter sem "Get"**: `Header()`, `Status()`, `Error()` (não `GetHeader()`)

### Variáveis e Campos
- **camelCase**: `inflight`, `connStatus`, `shortMessage`
- **Acrônimos uppercase quando no início**: `SMSCAddr` (não `SmscAddr`)
- **Descritivo**: `respTimeout` (não `rt`)

### Constantes
- **PascalCase**: `SubmitSMID`, `DeliverSMID`
- **Enum-like**: Agrupar constantes relacionadas
  ```go
  const (
      BindTransmitterID    = 0x00000002
      BindReceiverID       = 0x00000001
      BindTransceiverID    = 0x00000009
  )
  ```

### Arquivos
- **Lowercase com underscores**: `submit_sm.go`, `bind_transmitter.go`
- **Match tipo principal**: `transmitter.go` contém `type Transmitter`
- **Sufixo `_test`**: `transmitter_test.go` para testes

---

## Padrões de Tratamento de Erros

### Erros Customizados

**Definição**: [smpp/transmitter.go:27-30](../smpp/transmitter.go#L27-L30)
```go
var (
    ErrNotConnected   = errors.New("not connected")
    ErrNotBound       = errors.New("not bound")
    ErrTimeout        = errors.New("timeout waiting for response")
    ErrMaxWindowSize  = errors.New("max window size reached")
)
```

**Uso**:
```go
if err == smpp.ErrTimeout {
    // Trate timeout especificamente
}
```

### Erros do Protocolo (Status Codes)

**Tipo**: [smpp/pdu/pdu.go:154-158](../smpp/pdu/pdu.go#L154-L158)
```go
type Status uint32

func (s Status) Error() string {
    return fmt.Sprintf("SMPP Status: 0x%08x", uint32(s))
}
```

**Verificação**:
```go
resp, err := tx.Submit(sm)
if err != nil {
    if status, ok := err.(pdu.Status); ok {
        if status == pdu.InvalidDestinationAddress {
            // Trate número inválido
        }
    }
}
```

### Error Wrapping (Go 1.13+)

**Uso**: Adicionar contexto a erros
```go
if err := conn.Write(p); err != nil {
    return fmt.Errorf("failed to write PDU: %w", err)
}
```

**Verificação**:
```go
if errors.Is(err, net.ErrClosed) {
    // Conexão foi fechada
}
```

---

## Padrões de Teste

### Naming Convention
- Arquivo: `transmitter_test.go`
- Função: `TestTransmitter_Submit`
- Subtests: `t.Run("with valid message", func(t *testing.T) { ... })`

### Table-Driven Tests

**Exemplo**: [smpp/encoding/gsm7_test.go](../smpp/encoding/gsm7_test.go)
```go
tests := []struct{
    name  string
    input string
    want  []byte
}{
    {"ASCII", "Hello", []byte{0x48, 0x65, ...}},
    {"Extended", "€50", []byte{0x1B, 0x65, ...}},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := gsm7.Encode(tt.input)
        if !bytes.Equal(got, tt.want) {
            t.Errorf("got %v, want %v", got, tt.want)
        }
    })
}
```

### Mock Server para Testes

**Uso**: [smpp/smpptest/smpptest.go](../smpp/smpptest/smpptest.go)
```go
srv := smpptest.NewUnstartedServer(handler)
srv.Start()
defer srv.Close()

tx := &smpp.Transmitter{Addr: srv.Addr}
// Testes com servidor local
```

### Test Helpers

**Padrão**: Funções auxiliares com prefixo `test`
```go
func testTransmitter(t *testing.T) *smpp.Transmitter {
    // Setup comum
}
```

---

## Padrões de Concorrência

### Uso de Goroutines

**Regra**: Sempre que há I/O ou operação bloqueante
```go
go func() {
    for {
        p, err := conn.Read()
        if err != nil {
            return
        }
        handler(p)
    }
}()
```

### Sincronização com Channels

**Padrão**: Preferir channels a mutexes quando possível
```go
// Bom: channel para comunicação
done := make(chan struct{})
go work(done)
<-done

// Evitar: mutex para comunicação (use para proteção de dados)
```

### Context para Cancelamento

**Uso**: [smpp/transmitter.go:252-257](../smpp/transmitter.go#L252-L257)
```go
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error) {
    ctx := context.Background()
    if t.RespTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, t.RespTimeout)
        defer cancel()
    }
    // ...
}
```

### WaitGroup para Coordenação

**Padrão**: Aguardar múltiplas goroutines
```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        // work
    }()
}
wg.Wait()
```

---

## Boas Práticas Específicas

### 1. Zero-Value Usable

**Princípio**: Structs devem ser úteis com valores zero quando possível

**Exemplo**: [smpp/transmitter.go:40-50](../smpp/transmitter.go#L40-L50)
```go
tx := &smpp.Transmitter{
    Addr:   "localhost:2775",
    User:   "user",
    Passwd: "pass",
    // EnquireLink, RespTimeout têm defaults se não especificados
}
```

### 2. Accept Interfaces, Return Structs

**Padrão**: Funções aceitam interfaces, retornam tipos concretos
```go
// Aceita interface
func process(c Conn) error { ... }

// Retorna struct concreta
func NewTransmitter() *Transmitter { ... }
```

### 3. Method Receivers

**Regra**:
- **Pointer receiver**: se modifica estado OU struct é grande
- **Value receiver**: se não modifica estado E struct é pequena

```go
// Pointer: modifica estado
func (t *Transmitter) Submit(sm *ShortMessage) error { ... }

// Value: não modifica, retorna cópia
func (h Header) Len() int { return 16 }
```

### 4. Configuração via Struct Fields

**Padrão**: Evitar Options pattern, usar struct fields diretamente
```go
tx := &smpp.Transmitter{
    Addr:          "localhost:2775",
    User:          "user",
    Passwd:        "pass",
    EnquireLink:   10 * time.Second,
    RespTimeout:   5 * time.Second,
    RateLimiter:   myLimiter,
    Middleware:    loggingMiddleware,
}
```

**Benefício**: Simples, auto-documentado, sem funções adicionais

### 5. Documentação via Comentários

**Padrão**: Comentários em godoc format
```go
// Transmitter é um cliente SMPP para envio de SMS.
// Mantém conexão persistente com reconexão automática.
//
// Exemplo de uso:
//   tx := &Transmitter{Addr: "localhost:2775"}
//   tx.Bind()
//   tx.Submit(&ShortMessage{Dst: "5511999999999", Text: pdutext.Raw("Hi")})
type Transmitter struct { ... }
```

---

## Anti-Patterns Evitados

### ❌ God Objects
**Evitado**: Não há struct única que faz tudo
**Solução**: Responsabilidades divididas (Transmitter, Receiver, Transceiver)

### ❌ Singleton
**Evitado**: Sem package-level variables para estado
**Solução**: Factory pattern para sequence numbers

### ❌ Callback Hell
**Evitado**: Não há nested callbacks profundos
**Solução**: Handlers simples de um nível

### ❌ Premature Abstraction
**Evitado**: Interfaces apenas onde necessário
**Solução**: Interfaces criadas quando há múltiplas implementações

### ❌ Magic Numbers
**Evitado**: Constantes nomeadas para todos valores SMPP
**Solução**: [pdu/pdu.go:77-120](../smpp/pdu/pdu.go#L77-L120) define todas constantes

---

## Resumo de Padrões

| Padrão | Onde Usado | Benefício Principal |
|--------|-----------|---------------------|
| Layered Architecture | Toda biblioteca | Separação de concerns |
| Factory | Sequence numbers | Thread-safety sem globals |
| Strategy | Text encodings | Extensibilidade |
| Adapter | Connection middleware | Composição |
| Observer | Status notifications | Reatividade |
| Template Method | PDU types | Reuso de código |
| Connection Pool | Transmitter/Receiver | Resiliência |
| Request-Response Correlation | inflight map | Concorrência |

---

## Próximos Passos

Consulte:
- [**Stack Tecnológica**](stack.md) - Arquitetura geral e dependências
- [**Funcionalidades**](features.md) - Como usar cada feature
- [**Regras de Negócio**](business-rules.md) - Protocolo SMPP
- [**Integração**](integrations.md) - Usar em seus projetos
