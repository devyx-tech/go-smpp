# Funcionalidades

Este documento descreve em detalhes as funcionalidades principais da biblioteca **go-smpp**.

---

## Funcionalidades Principais

### 1. Transmitter - Envio de SMS

**Descrição**: Cliente SMPP para **envio** de mensagens SMS. Mantém conexão persistente com SMSC com reconexão automática.

**Localização**: [smpp/transmitter.go](../smpp/transmitter.go)

#### Características

**Conexão Persistente**:
- Estabelece conexão TCP/TLS com SMSC
- Envia Bind Transmitter PDU para autenticação
- Mantém conexão ativa com EnquireLink keepalive

**Reconexão Automática**:
- Detecta falhas de conexão automaticamente
- Reconecta com exponential backoff (1s → 2s → 4s → ... → max 120s)
- Notifica status via channel `<-chan ConnStatus`

**Rate Limiting Opcional**:
- Suporta interface `RateLimiter` para controle de throughput
- Integração com `golang.org/x/time/rate`

**TLS Support**:
- Configurável via campo `TLS` (tipo `*tls.Config`)
- Validação de certificados customizável

#### Configuração

```go
type Transmitter struct {
    Addr        string              // Endereço SMSC (ex: "localhost:2775")
    User        string              // SystemID para autenticação
    Passwd      string              // Password
    SystemType  string              // Tipo de sistema (opcional)
    EnquireLink time.Duration       // Intervalo de keepalive (padrão: 10s)
    RespTimeout time.Duration       // Timeout de resposta (padrão: 1s)
    WindowSize  int                 // Max requests concorrentes (0 = ilimitado)
    RateLimiter RateLimiter         // Rate limiter opcional
    TLS         *tls.Config         // Configuração TLS opcional
    Middleware  ConnMiddleware      // Middleware de conexão opcional
}
```

#### Uso Básico

```go
// Criar transmitter
tx := &smpp.Transmitter{
    Addr:   "smsc.example.com:2775",
    User:   "username",
    Passwd: "password",
}

// Bind (conectar)
conn := tx.Bind()
status := <-conn  // Aguarda conexão

if status.Error() != nil {
    log.Fatal("Erro ao conectar:", status.Error())
}

// Enviar SMS simples
sm := &smpp.ShortMessage{
    Src:  "1234",              // Número origem
    Dst:  "5511999999999",     // Número destino
    Text: pdutext.Raw("Olá!"), // Texto
}

resp, err := tx.Submit(sm)
if err != nil {
    log.Fatal("Erro ao enviar:", err)
}

log.Printf("MessageID: %s", resp.MessageID)

// Fechar conexão
tx.Close()
```

#### Envio com Opções Avançadas

```go
sm := &smpp.ShortMessage{
    Src:             "1234",
    Dst:             "5511999999999",
    Text:            pdutext.GSM7("Mensagem com acentuação"),
    Validity:        24 * time.Hour,                    // Validade de 24h
    Register:        smpp.FinalDeliveryReceipt,         // Solicitar delivery receipt
    ServiceType:     "SMS",                             // Tipo de serviço
    SourceAddrTON:   5,                                 // Type of Number
    SourceAddrNPI:   0,                                 // Numbering Plan Indicator
    DestAddrTON:     1,                                 // International
    DestAddrNPI:     1,                                 // ISDN
}

resp, err := tx.Submit(sm)
```

#### Envio para Múltiplos Destinos

```go
sm := &smpp.ShortMessage{
    Src:     "1234",
    DstList: []string{"5511999999999", "5511888888888", "5511777777777"},
    Text:    pdutext.Raw("Mensagem em massa"),
}

resp, err := tx.Submit(sm)
// Retorna SubmitMultiResp com lista de MessageIDs
```

#### Monitoramento de Status de Conexão

```go
conn := tx.Bind()

go func() {
    for status := range conn {
        switch status.Status() {
        case smpp.Connected:
            log.Println("✓ Conectado ao SMSC")
        case smpp.Disconnected:
            log.Println("✗ Desconectado. Reconectando...")
        }

        if status.Error() != nil {
            log.Println("Erro:", status.Error())
        }
    }
}()
```

#### Componentes Envolvidos
- [transmitter.go:40-76](../smpp/transmitter.go#L40-L76): Struct e configuração
- [transmitter.go:78-135](../smpp/transmitter.go#L78-L135): Método `Bind()`
- [transmitter.go:240-335](../smpp/transmitter.go#L240-L335): Método `Submit()`
- [client.go](../smpp/client.go): Lógica de reconexão
- [pdu/submit_sm.go](../smpp/pdu/submit_sm.go): PDU de submissão

#### Dependências
- Conexão de rede ao SMSC
- Credenciais válidas (User/Passwd)
- Porta SMPP acessível (geralmente 2775 ou 2776 para TLS)

---

### 2. Receiver - Recebimento de SMS

**Descrição**: Cliente SMPP para **recebimento** de mensagens SMS. Processa DeliverSM PDUs e faz merge automático de mensagens longas.

**Localização**: [smpp/receiver.go](../smpp/receiver.go)

#### Características

**Handler-Based Processing**:
- Callback `HandlerFunc` chamado para cada PDU recebido
- Resposta automática com DeliverSMResp

**Long Message Merging**:
- Detecta mensagens longas via UDH (User Data Header)
- Monta mensagem completa a partir de múltiplas partes
- Cleanup automático de partes incompletas após timeout

**Reconexão Automática**:
- Mesmas capacidades do Transmitter
- Mantém handler ativo durante reconexões

#### Configuração

```go
type Receiver struct {
    Addr                 string           // Endereço SMSC
    User                 string           // SystemID
    Passwd               string           // Password
    SystemType           string           // Tipo de sistema (opcional)
    EnquireLink          time.Duration    // Intervalo keepalive (padrão: 10s)
    Handler              HandlerFunc      // Callback para PDUs recebidos
    TLS                  *tls.Config      // TLS opcional
    LongMessageMerge     bool             // Habilitar merge (padrão: true)
    MergeInterval        time.Duration    // Intervalo de merge (padrão: 1s)
    MergeCleanupInterval time.Duration    // Cleanup de partes (padrão: 5min)
}
```

#### Uso Básico

```go
rx := &smpp.Receiver{
    Addr:   "smsc.example.com:2775",
    User:   "username",
    Passwd: "password",
    Handler: func(p pdu.Body) {
        switch p.Header().ID {
        case pdu.DeliverSMID:
            fields := p.Fields()

            src := fields[pdufield.SourceAddr]
            dst := fields[pdufield.DestinationAddr]
            text := fields[pdufield.ShortMessage]

            log.Printf("SMS de %s para %s: %s", src, dst, text)
        }
    },
}

conn := rx.Bind()
<-conn  // Aguarda conexão

// Mantém receiver rodando
select {}  // ou use um canal de shutdown
```

#### Handler Avançado com Type Assertions

```go
Handler: func(p pdu.Body) {
    switch p.Header().ID {
    case pdu.DeliverSMID:
        if deliverSM, ok := p.(*pdu.DeliverSM); ok {
            // Acesso tipado aos campos
            processDeliverSM(deliverSM)
        }

    case pdu.DataSMID:
        // Processar DataSM (se implementado)
    }
}
```

#### Long Message Handling

```go
rx := &smpp.Receiver{
    Addr:                 "smsc.example.com:2775",
    User:                 "username",
    Passwd:               "password",
    LongMessageMerge:     true,              // Habilitar merge (padrão)
    MergeInterval:        2 * time.Second,   // Verificar a cada 2s
    MergeCleanupInterval: 10 * time.Minute,  // Limpar após 10min
    Handler: func(p pdu.Body) {
        fields := p.Fields()
        text := fields[pdufield.ShortMessage]

        // 'text' já contém a mensagem completa montada!
        log.Printf("Mensagem: %s (len=%d)", text, len(text))
    },
}
```

**Como funciona**:
1. Receiver detecta UDH header em DeliverSM
2. Extrai reference number, total parts, current part
3. Armazena partes em map interno
4. Quando todas partes chegam, monta mensagem completa
5. Chama Handler com mensagem montada
6. Cleanup remove partes incompletas após `MergeCleanupInterval`

#### Componentes Envolvidos
- [receiver.go:40-72](../smpp/receiver.go#L40-L72): Struct e configuração
- [receiver.go:74-115](../smpp/receiver.go#L74-L115): Método `Bind()`
- [receiver.go:145-220](../smpp/receiver.go#L145-L220): Lógica de merge
- [pdu/deliver_sm.go](../smpp/pdu/deliver_sm.go): PDU de entrega

---

### 3. Transceiver - Envio e Recebimento Simultâneo

**Descrição**: Cliente SMPP **bidirecional** que combina Transmitter e Receiver em uma única conexão.

**Localização**: [smpp/transceiver.go](../smpp/transceiver.go)

#### Características

**Single Connection**:
- Uma conexão para envio E recebimento
- Economia de recursos (apenas 1 socket TCP)
- Alguns SMSCs exigem Transceiver

**Funcionalidades Combinadas**:
- Método `Submit()` para envio (como Transmitter)
- Handler para recebimento (como Receiver)
- Long message merging

#### Configuração

```go
type Transceiver struct {
    Addr                 string
    User                 string
    Passwd               string
    SystemType           string
    EnquireLink          time.Duration
    RespTimeout          time.Duration
    WindowSize           int
    Handler              HandlerFunc       // Para PDUs recebidos
    RateLimiter          RateLimiter
    TLS                  *tls.Config
    LongMessageMerge     bool
    MergeInterval        time.Duration
    MergeCleanupInterval time.Duration
}
```

#### Uso

```go
tc := &smpp.Transceiver{
    Addr:   "smsc.example.com:2775",
    User:   "username",
    Passwd: "password",
    Handler: func(p pdu.Body) {
        // Processar mensagens recebidas
        log.Printf("PDU recebido: %v", p.Header().ID)
    },
}

conn := tc.Bind()
<-conn

// Enviar SMS
resp, err := tc.Submit(&smpp.ShortMessage{
    Src:  "1234",
    Dst:  "5511999999999",
    Text: pdutext.Raw("Hello"),
})

// Receber via Handler
```

#### Quando Usar
- **Use Transceiver**: Quando precisa enviar e receber simultaneamente
- **Use Transmitter + Receiver**: Quando SMSC exige conexões separadas ou para melhor isolamento

#### Componentes Envolvidos
- [transceiver.go](../smpp/transceiver.go): Implementação completa

---

### 4. Query - Consulta de Status de Mensagem

**Descrição**: Consultar o status de entrega de uma mensagem enviada anteriormente.

**Localização**: [transmitter.go:337-370](../smpp/transmitter.go#L337-L370)

#### Uso

```go
// Primeiro, enviar mensagem
resp, _ := tx.Submit(&smpp.ShortMessage{
    Src:  "1234",
    Dst:  "5511999999999",
    Text: pdutext.Raw("Test"),
})

messageID := resp.MessageID

// Depois, consultar status
time.Sleep(10 * time.Second)  // Aguardar processamento

queryResp, err := tx.QuerySM(messageID, "5511999999999")
if err != nil {
    log.Fatal(err)
}

log.Printf("Message State: %s", queryResp.MessageState)
log.Printf("Error Code: %d", queryResp.ErrorCode)
```

#### Estados Possíveis

```go
const (
    SCHEDULED      MessageState = 0  // Agendada
    ENROUTE        MessageState = 1  // Em rota
    DELIVERED      MessageState = 2  // Entregue
    EXPIRED        MessageState = 3  // Expirada
    DELETED        MessageState = 4  // Deletada
    UNDELIVERABLE  MessageState = 5  // Não entregável
    ACCEPTED       MessageState = 6  // Aceita
    UNKNOWN        MessageState = 7  // Desconhecida
    REJECTED       MessageState = 8  // Rejeitada
)
```

#### Componentes Envolvidos
- [transmitter.go:337-370](../smpp/transmitter.go#L337-L370): Método `QuerySM()`
- [pdu/query_sm.go](../smpp/pdu/query_sm.go): PDU de query

---

### 5. Text Encodings - Codificação de Caracteres

**Descrição**: Múltiplos codecs para suporte a diferentes character sets.

**Localização**: [smpp/pdu/pdutext/](../smpp/pdu/pdutext/)

#### Codecs Disponíveis

##### 5.1 GSM7 (GSM 03.38)

**Uso**: Encoding padrão para SMS, 160 caracteres por mensagem

**Localização**: [pdutext/gsm7.go](../smpp/pdu/pdutext/gsm7.go), [encoding/gsm7.go](../smpp/encoding/gsm7.go)

```go
text := pdutext.GSM7("Hello World!")  // Encoding básico

// Caracteres estendidos (com escape 0x1B)
text := pdutext.GSM7("Custo: €10")    // € requer escape

// Packed format (mais eficiente)
text := pdutext.GSM7Packed("Message")
```

**Caracteres Suportados**:
- A-Z, a-z, 0-9
- Símbolos comuns: @£$¥èéùìòÇ
- Caracteres estendidos (via escape): €[]{}^\|~

**Limite**: 160 caracteres (ou 153 com UDH para mensagens longas)

##### 5.2 Latin1 (ISO-8859-1)

**Uso**: Caracteres ocidentais (português, espanhol, francês, etc.)

**Localização**: [pdutext/latin1.go](../smpp/pdu/pdutext/latin1.go)

```go
text := pdutext.Latin1("Olá! Acentuação completa: áéíóú")
```

**Limite**: 70 caracteres (ou 67 com UDH)

##### 5.3 UCS2 (Unicode)

**Uso**: Caracteres internacionais (chinês, árabe, emoji, etc.)

**Localização**: [pdutext/ucs2.go](../smpp/pdu/pdutext/ucs2.go)

```go
text := pdutext.UCS2("你好世界")         // Chinês
text := pdutext.UCS2("مرحبا بالعالم")  // Árabe
text := pdutext.UCS2("Hello 😊")       // Emoji
```

**Encoding**: UTF-16 Big-Endian

**Limite**: 70 caracteres (ou 67 com UDH)

##### 5.4 ISO-8859-5 (Cyrillic)

**Uso**: Alfabeto cirílico (russo, ucraniano, búlgaro, etc.)

**Localização**: [pdutext/iso88595.go](../smpp/pdu/pdutext/iso88595.go)

```go
text := pdutext.ISO88595("Привет мир")  // Russo
```

**Limite**: 70 caracteres

##### 5.5 Raw (Bytes Brutos)

**Uso**: Envio de bytes sem transformação

**Localização**: [pdutext/raw.go](../smpp/pdu/pdutext/raw.go)

```go
text := pdutext.Raw("Plain ASCII text")
text := pdutext.Raw([]byte{0x48, 0x65, 0x6C, 0x6C, 0x6F})
```

**Limite**: Depende do encoding especificado no DataCoding

#### Comparação de Encodings

| Encoding | Caracteres/SMS | UDH Limit | Casos de Uso |
|----------|---------------|-----------|--------------|
| GSM7     | 160           | 153       | Inglês, números, símbolos básicos |
| GSM7Packed | 160         | 153       | GSM7 com compressão |
| Latin1   | 70            | 67        | Português, espanhol, francês |
| UCS2     | 70            | 67        | Chinês, árabe, emoji, multilingue |
| ISO88595 | 70            | 67        | Russo, ucraniano, búlgaro |
| Raw      | Variável      | Variável  | Binário, WAP Push, casos especiais |

#### Seleção Automática de Encoding

```go
func chooseEncoding(text string) pdutext.Codec {
    // Se contém apenas caracteres GSM7, usar GSM7
    if isGSM7(text) {
        return pdutext.GSM7(text)
    }

    // Se contém acentos latinos, usar Latin1
    if isLatin1(text) {
        return pdutext.Latin1(text)
    }

    // Caso contrário, usar UCS2 (suporta tudo)
    return pdutext.UCS2(text)
}
```

---

### 6. Long Messages - Mensagens Longas

**Descrição**: Splitting e merging automático de mensagens que excedem limites de SMS único.

#### Envio de Mensagens Longas (Transmitter)

**Método 1: Automático via `SubmitLongMsg()`**

**Localização**: [transmitter.go:372-450](../smpp/transmitter.go#L372-L450)

```go
longText := strings.Repeat("A", 200)  // 200 caracteres

resps, err := tx.SubmitLongMsg(&smpp.ShortMessage{
    Src:  "1234",
    Dst:  "5511999999999",
    Text: pdutext.GSM7(longText),
})

// Retorna slice de SubmitSMResp (uma para cada parte)
for i, resp := range resps {
    log.Printf("Parte %d - MessageID: %s", i+1, resp.MessageID)
}
```

**Como funciona**:
1. Detecta que mensagem excede limite (153 chars para GSM7, 67 para UCS2)
2. Adiciona UDH (User Data Header) com metadados:
   - IEI (Information Element Identifier): 0x00
   - IEDL (Information Element Data Length): 0x03
   - Reference Number: número único para agrupar partes
   - Total Parts: número total de partes
   - Current Part: número da parte atual (1-indexed)
3. Divide texto em partes
4. Envia cada parte como SubmitSM separado
5. Retorna array de responses

**Método 2: Manual**

```go
// Dividir manualmente e adicionar UDH
parts := splitMessage(longText, 153)
refNum := uint8(rand.Intn(255))

for i, part := range parts {
    sm := &smpp.ShortMessage{
        Src:  "1234",
        Dst:  "5511999999999",
        Text: pdutext.GSM7(part),
        UDH: []byte{
            0x00,                    // IEI: Concatenated SMS
            0x03,                    // IEDL: 3 bytes
            refNum,                  // Reference number
            uint8(len(parts)),       // Total parts
            uint8(i + 1),            // Current part (1-indexed)
        },
    }

    tx.Submit(sm)
}
```

#### Recebimento de Mensagens Longas (Receiver)

**Automático com Merge**:

```go
rx := &smpp.Receiver{
    Addr:                 "smsc.example.com:2775",
    User:                 "username",
    Passwd:               "password",
    LongMessageMerge:     true,              // Padrão: true
    MergeInterval:        1 * time.Second,   // Verificar a cada 1s
    MergeCleanupInterval: 5 * time.Minute,   // Limpar após 5min
    Handler: func(p pdu.Body) {
        fields := p.Fields()
        text := fields[pdufield.ShortMessage]

        // 'text' já é a mensagem completa!
        log.Printf("Mensagem completa: %s", text)
    },
}
```

**Funcionamento Interno**: [receiver.go:145-220](../smpp/receiver.go#L145-L220)
1. Detecta UDH em DeliverSM
2. Extrai reference number, total parts, current part
3. Armazena parte em `map[refNum]map[partNum]part`
4. Timer periódico verifica se todas partes chegaram
5. Se completo, monta mensagem e chama Handler
6. Cleanup remove mensagens incompletas após timeout

**Desabilitar Merge** (receber partes individualmente):

```go
rx := &smpp.Receiver{
    // ...
    LongMessageMerge: false,
    Handler: func(p pdu.Body) {
        // Recebe cada parte separadamente
        fields := p.Fields()
        text := fields[pdufield.ShortMessage]

        // Processar UDH manualmente se necessário
    },
}
```

---

### 7. CLI Tools - Ferramentas de Linha de Comando

#### 7.1 sms - Cliente SMS CLI

**Descrição**: Cliente CLI para envio de SMS e consulta de status.

**Localização**: [cmd/sms/](../cmd/sms/)

**Instalação**:
```bash
go install github.com/devyx-tech/go-smpp/cmd/sms@latest
```

**Uso - Enviar SMS**:
```bash
sms submit \
  --addr=smsc.example.com:2775 \
  --user=username \
  --passwd=password \
  --src=1234 \
  --dst=5511999999999 \
  --text="Hello World"
```

**Uso - Consultar Status**:
```bash
sms query \
  --addr=smsc.example.com:2775 \
  --user=username \
  --passwd=password \
  --msgid=abc123 \
  --src=1234 \
  --dst=5511999999999
```

**Flags Disponíveis**:
- `--addr`: Endereço SMSC
- `--user`: SystemID
- `--passwd`: Password
- `--src`: Número origem
- `--dst`: Número destino
- `--text`: Texto da mensagem
- `--msgid`: MessageID (para query)
- `--encoding`: Encoding (gsm7, latin1, ucs2)
- `--validity`: Período de validade

#### 7.2 smsapid - SMS API Daemon

**Descrição**: Daemon HTTP que expõe API REST para envio de SMS via SMPP.

**Localização**: [cmd/smsapid/](../cmd/smsapid/)

**Instalação**:
```bash
go install github.com/devyx-tech/go-smpp/cmd/smsapid@latest
```

**Uso**:
```bash
smsapid \
  --addr=smsc.example.com:2775 \
  --user=username \
  --passwd=password \
  --http=:8080
```

**API Endpoints**:

**POST /sms**:
```bash
curl -X POST http://localhost:8080/sms \
  -H "Content-Type: application/json" \
  -d '{
    "src": "1234",
    "dst": "5511999999999",
    "text": "Hello via API"
  }'
```

**GET /health**:
```bash
curl http://localhost:8080/health
```

---

## Funcionalidades Secundárias

### 8. Rate Limiting

**Descrição**: Controle de throughput para respeitar limites do SMSC.

**Interface**: [transmitter.go:38](../smpp/transmitter.go#L38)

```go
type RateLimiter interface {
    Wait(ctx context.Context) error
}
```

**Uso com golang.org/x/time/rate**:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 1)  // 10 msgs/segundo, burst de 1

tx := &smpp.Transmitter{
    Addr:        "smsc.example.com:2775",
    User:        "username",
    Passwd:      "password",
    RateLimiter: limiter,
}

// Submit() automaticamente aguarda rate limiter
tx.Submit(&smpp.ShortMessage{...})  // Rate limited
```

**Localização**: [transmitter.go:252-257](../smpp/transmitter.go#L252-L257)

---

### 9. Connection Middleware

**Descrição**: Wrapper de conexões para logging, metrics, tracing.

**Interface**: [client.go:34-36](../smpp/client.go#L34-L36)

```go
type ConnMiddleware func(Conn) Conn
```

**Exemplo - Logging Middleware**:

```go
func loggingMiddleware(next smpp.Conn) smpp.Conn {
    return &loggingConn{next: next}
}

type loggingConn struct {
    next smpp.Conn
}

func (c *loggingConn) Write(p pdu.Body) error {
    log.Printf("→ Enviando PDU: %v", p.Header().ID)
    return c.next.Write(p)
}

func (c *loggingConn) Read() (pdu.Body, error) {
    p, err := c.next.Read()
    if err == nil {
        log.Printf("← Recebido PDU: %v", p.Header().ID)
    }
    return p, err
}

func (c *loggingConn) Close() error {
    return c.next.Close()
}

// Uso
tx := &smpp.Transmitter{
    Addr:       "smsc.example.com:2775",
    Middleware: loggingMiddleware,
}
```

---

### 10. Test Server (smpptest)

**Descrição**: Servidor SMPP in-process para testes unitários.

**Localização**: [smpp/smpptest/](../smpp/smpptest/)

**Uso**:

```go
import "github.com/devyx-tech/go-smpp/smpp/smpptest"

func TestMyCode(t *testing.T) {
    // Criar servidor de teste
    srv := smpptest.NewUnstartedServer(func(c smpp.Conn) {
        // Handler do servidor
        for {
            p, err := c.Read()
            if err != nil {
                return
            }

            // Responder a SubmitSM
            if p.Header().ID == pdu.SubmitSMID {
                resp := &pdu.SubmitSMResp{
                    // ...
                }
                c.Write(resp)
            }
        }
    })

    srv.Start()
    defer srv.Close()

    // Usar servidor em teste
    tx := &smpp.Transmitter{Addr: srv.Addr}
    tx.Bind()
    // ...
}
```

---

## Funcionalidades em Desenvolvimento (TODO)

As seguintes funcionalidades estão **planejadas mas não implementadas**:

### PDU Types Not Implemented

**Localização**: [pdu/pdu.go:120-127](../smpp/pdu/pdu.go#L120-L127)

- **outbind**: Server-initiated bind
- **alert_notification**: Alertas de disponibilidade
- **data_sm / data_sm_resp**: Transferência de dados alternativa
- **cancel_sm / cancel_sm_resp**: Cancelamento de mensagem agendada
- **replace_sm / replace_sm_resp**: Substituição de mensagem agendada

**Razão**: Pouco usados na prática; maioria dos SMSCs não suporta.

---

## Resumo de Funcionalidades

| Funcionalidade | Status | Localização | Prioridade |
|---------------|--------|-------------|------------|
| Transmitter (envio) | ✅ Completo | transmitter.go | Alta |
| Receiver (recebimento) | ✅ Completo | receiver.go | Alta |
| Transceiver (bidirecional) | ✅ Completo | transceiver.go | Alta |
| Query (status de msg) | ✅ Completo | transmitter.go | Média |
| GSM7 Encoding | ✅ Completo | pdutext/gsm7.go | Alta |
| Latin1 Encoding | ✅ Completo | pdutext/latin1.go | Média |
| UCS2 Encoding | ✅ Completo | pdutext/ucs2.go | Média |
| ISO-8859-5 Encoding | ✅ Completo | pdutext/iso88595.go | Baixa |
| Long Messages (split) | ✅ Completo | transmitter.go | Alta |
| Long Messages (merge) | ✅ Completo | receiver.go | Alta |
| Rate Limiting | ✅ Completo | transmitter.go | Média |
| TLS Support | ✅ Completo | client.go | Média |
| Middleware | ✅ Completo | client.go | Baixa |
| CLI tools | ✅ Completo | cmd/ | Baixa |
| Test Server | ✅ Completo | smpptest/ | Média |
| AlertNotification | ❌ TODO | - | Baixa |
| DataSM | ❌ TODO | - | Baixa |
| CancelSM | ❌ TODO | - | Baixa |
| ReplaceSM | ❌ TODO | - | Baixa |

---

## Próximos Passos

Consulte:
- [**Regras de Negócio**](business-rules.md) - Detalhes do protocolo SMPP
- [**Integração**](integrations.md) - Como usar em seus projetos
- [**Stack**](stack.md) - Arquitetura e dependências
- [**Padrões**](patterns.md) - Design patterns utilizados
