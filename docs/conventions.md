# Convenções de Código

**Data de Análise:** 2026-03-24

> Estas convenções cobrem **apenas padrões não aplicados por linters/formatters**.
> Não há linter configurado (sem `.golangci.yml`) — use `go vet` e `gofmt` como baseline.

## Naming (além do que gofmt enforce)

**Pacotes:** Use nomes curtos, lowercase, sem underscores. Prefixe sub-pacotes com o nome do pai quando necessário (`pdufield`, `pdutext`, `pdutlv`).

**Interfaces:** Use nomes substantivos sem prefixo `I`. Interfaces de uma operação recebem sufixo `-er` (`Reader`, `Writer`, `Closer`).

**Constructors:** Use `New` + tipo para PDUs públicos (`NewSubmitSM()`, `NewBindTransmitter()`). Para constructors com sequence number, adicione sufixo `Seq` (`NewDeliverSMRespSeq(seq)`).

**Constantes de PDU ID:** Use sufixo `ID` (`SubmitSMID`, `BindTransmitterRespID`).

**Campos PDU:** Use nomes SMPP em snake_case como `pdufield.Name` string constants (`source_addr`, `data_coding`).

**Constantes de Tag TLV:** Use CamelCase descritivo (`ReceiptedMessageID`, `UserMessageReference`).

## Padrões de Código

### Composição via Embedding

**Quando usar:** Para combinar funcionalidades de tipos existentes sem duplicar código.
**Regra:** Use embedding de struct para composição. O `Transceiver` embute `Transmitter` para reutilizar toda a lógica de envio.

```go
// Extraído de smpp/transceiver.go:20-36
type Transceiver struct {
    Addr               string
    User               string
    Passwd             string
    SystemType         string
    EnquireLink        time.Duration
    EnquireLinkTimeout time.Duration
    RespTimeout        time.Duration
    BindInterval       time.Duration
    TLS                *tls.Config
    Handler            HandlerFunc
    ConnInterceptor    ConnMiddleware
    RateLimiter        RateLimiter
    WindowSize         uint

    Transmitter
}
```

### PDU Factory Pattern

**Quando usar:** Ao criar novos tipos de PDU.
**Regra:** Cada tipo de PDU segue o padrão: struct com `*Codec` embeddado + constructor privado `newXxx(hdr)` + constructor público `NewXxx()`. O constructor privado define a lista de campos; o público define o header ID e chama `init()`.

```go
// Extraído de smpp/pdu/types.go:196-231
func newSubmitSM(hdr *Header) *Codec {
    return &Codec{
        h: hdr,
        l: pdufield.List{
            pdufield.ServiceType,
            pdufield.SourceAddrTON,
            pdufield.SourceAddrNPI,
            pdufield.SourceAddr,
            // ... campos restantes
        },
    }
}

func NewSubmitSM() Body {
    b := newSubmitSM(&Header{ID: SubmitSMID})
    b.init()
    return b
}
```

### Tratamento de Erros — Sentinelas e Status

**Quando usar:** Erros de estado de conexão usam sentinelas (`var Err...`). Erros de resposta SMPP retornam `pdu.Status` diretamente (implementa `error`).
**Regra:** Nunca crie novos tipos de erro para condições cobertas por `pdu.Status`. Use sentinelas apenas para erros de conexão/estado. Retorne `pdu.Status` como erro quando o SMSC responde com status != 0.

```go
// Extraído de smpp/transmitter.go:439-445
if id := resp.PDU.Header().ID; id != pdu.SubmitSMRespID {
    return sm, fmt.Errorf("unexpected PDU ID: %s", id)
}
if s := resp.PDU.Header().Status; s != 0 {
    return sm, s
}
```

### ConnMiddleware como Decorator

**Quando usar:** Para interceptar tráfego SMPP (logging, métricas, debugging).
**Regra:** Implemente `ConnMiddleware` como uma função que wrappa `Conn` e retorna um novo `Conn`. Use somente para observação — não altere o tráfego.

```go
// Extraído de smpp/client.go:74
type ConnMiddleware func(conn Conn) Conn
```

### Connection Status via Channel

**Quando usar:** Para notificar mudanças de estado da conexão.
**Regra:** `Bind()` retorna `<-chan ConnStatus`. Use select sobre o channel para reagir. O channel é fechado quando `Close()` é chamado.

```go
// Extraído de smpp/example_test.go:38-45
conn := r.Bind()
time.AfterFunc(10*time.Second, func() { r.Close() })
for c := range conn {
    log.Println("SMPP connection status:", c.Status())
}
```

### Server Handlers

**Quando usar:** Ao implementar lógica de servidor SMPP.
**Regra:** Registre handlers por PDU ID usando `Handle(id, func)`. Para autenticação customizada, use `HandleAuth(id, func)`. Sempre responda com o mesmo sequence number do request.

```go
// Extraído de smpp/server.go:297-309 (padrão de echo handler)
func EchoHandler(s Session, m pdu.Body) {
    s.Write(m)
}
```

## Organização de Imports

**Ordem obrigatória:**
1. Pacotes da stdlib (`crypto/tls`, `fmt`, `sync`, `time`)
2. Pacotes externos (`golang.org/x/...`, `github.com/urfave/cli`)
3. Pacotes internos do projeto (`github.com/devyx-tech/go-smpp/smpp/...`)

```go
// Extraído de smpp/transmitter.go:7-22
import (
    "crypto/tls"
    "encoding/binary"
    "errors"
    "fmt"
    "math/rand"
    "strconv"
    "sync"
    "sync/atomic"
    "time"

    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutlv"
)
```

## Padrão de Concorrência

**Regra:** Use `sync.Mutex` para proteger state compartilhado. Use channels para comunicação entre goroutines. Use `sync/atomic` para contadores simples. Use `sync.Once` para operações idempotentes como `Close()`.

```go
// Extraído de smpp/client.go:252-264
func (c *client) Close() error {
    c.once.Do(func() {
        close(c.stop)
        if err := c.conn.Write(pdu.NewUnbind()); err == nil {
            select {
            case <-c.inbox:
            case <-time.After(time.Second):
            }
        }
        c.conn.Close()
    })
    return nil
}
```

## Anti-Padrões

- **Não acesse campos de struct protegidos por mutex sem lock** — `connSwitch` e `inflight` map exigem lock antes de qualquer acesso
- **Não crie goroutines sem mecanismo de shutdown** — use channels `stop` ou `chanClose` para sinalizar término
- **Não use `pdu.NewXxx()` com sequence number manual** — use `NewXxxSeq(seq)` quando precisar definir seq, ou deixe o auto-increment do `Codec.init()`

---

*Análise de convenções: 2026-03-24*
