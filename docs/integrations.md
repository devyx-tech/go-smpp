# Guia de Integração

Este documento explica como integrar a biblioteca **go-smpp** em seus projetos Go para enviar e receber mensagens SMS via protocolo SMPP.

---

## Instalação

### Requisitos

- **Go 1.18+**
- Acesso de rede ao SMSC (Short Message Service Center)
- Credenciais SMPP (SystemID e Password)

### Adicionar Dependência

```bash
go get github.com/devyx-tech/go-smpp
```

Ou adicione ao `go.mod`:
```go
require github.com/devyx-tech/go-smpp v0.0.0-latest
```

Depois execute:
```bash
go mod download
```

---

## Import

```go
import (
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)
```

**Pacotes principais**:
- `smpp`: Cliente (Transmitter, Receiver, Transceiver)
- `smpp/pdu`: Tipos de PDU e estruturas
- `smpp/pdu/pdufield`: Campos de PDU
- `smpp/pdu/pdutext`: Codecs de texto (GSM7, UCS2, etc.)

---

## Casos de Uso Comuns

### 1. Enviar SMS Simples

**Cenário**: Serviço de notificações enviando SMS transacionais.

```go
package main

import (
    "log"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    // Configurar transmitter
    tx := &smpp.Transmitter{
        Addr:   "smsc.example.com:2775",
        User:   "your_username",
        Passwd: "your_password",
    }

    // Conectar
    conn := tx.Bind()
    status := <-conn  // Aguardar conexão

    if status.Error() != nil {
        log.Fatal("Erro ao conectar:", status.Error())
    }

    log.Println("Conectado ao SMSC!")

    // Enviar SMS
    resp, err := tx.Submit(&smpp.ShortMessage{
        Src:  "MyApp",                    // Remetente (pode ser alfanumérico)
        Dst:  "5511999999999",            // Destinatário
        Text: pdutext.Raw("Seu código de verificação é: 123456"),
    })

    if err != nil {
        log.Fatal("Erro ao enviar SMS:", err)
    }

    log.Printf("SMS enviado! MessageID: %s", resp.MessageID)

    // Fechar conexão ao terminar
    defer tx.Close()
}
```

---

### 2. Receber SMS (Webhook Style)

**Cenário**: Serviço que processa SMS recebidos (ex: respostas de clientes).

```go
package main

import (
    "log"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
)

func main() {
    // Configurar receiver
    rx := &smpp.Receiver{
        Addr:   "smsc.example.com:2775",
        User:   "your_username",
        Passwd: "your_password",
        Handler: processSMS,  // Callback para SMS recebidos
    }

    // Conectar
    conn := rx.Bind()
    status := <-conn

    if status.Error() != nil {
        log.Fatal("Erro ao conectar:", status.Error())
    }

    log.Println("Receiver ativo, aguardando SMS...")

    // Manter receiver rodando
    select {}  // Block forever
}

func processSMS(p pdu.Body) {
    // Processar apenas DeliverSM (SMS recebido)
    if p.Header().ID != pdu.DeliverSMID {
        return
    }

    fields := p.Fields()

    src := fields[pdufield.SourceAddr]
    dst := fields[pdufield.DestinationAddr]
    text := fields[pdufield.ShortMessage]

    log.Printf("SMS recebido de %s para %s: %s", src, dst, text)

    // Processar mensagem (salvar em BD, enviar para fila, etc.)
    go handleIncomingSMS(src.(string), dst.(string), text.([]byte))
}

func handleIncomingSMS(from, to string, message []byte) {
    // Sua lógica de negócio aqui
    log.Printf("Processando mensagem de %s: %s", from, string(message))
}
```

---

### 3. Enviar SMS com Delivery Receipt

**Cenário**: Rastrear se SMS foi entregue ao destinatário.

```go
package main

import (
    "log"
    "time"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    // Usar Transceiver para enviar E receber (DLRs)
    tc := &smpp.Transceiver{
        Addr:    "smsc.example.com:2775",
        User:    "your_username",
        Passwd:  "your_password",
        Handler: processDLR,  // Callback para delivery receipts
    }

    conn := tc.Bind()
    <-conn

    log.Println("Conectado! Enviando SMS...")

    // Enviar SMS com delivery receipt
    resp, err := tc.Submit(&smpp.ShortMessage{
        Src:      "MyApp",
        Dst:      "5511999999999",
        Text:     pdutext.Raw("Mensagem importante"),
        Register: smpp.FinalDeliveryReceipt,  // Solicitar DLR
        Validity: 24 * time.Hour,              // Válido por 24h
    })

    if err != nil {
        log.Fatal("Erro:", err)
    }

    log.Printf("SMS enviado! MessageID: %s - Aguardando DLR...", resp.MessageID)

    // Manter rodando para receber DLRs
    select {}
}

func processDLR(p pdu.Body) {
    if p.Header().ID != pdu.DeliverSMID {
        return
    }

    fields := p.Fields()
    esmClass := fields[pdufield.ESMClass].(uint8)

    // Verificar se é DLR (bit 2 do esm_class)
    if esmClass&0x04 != 0 {
        messageID := fields[pdufield.ReceiptedMessageID]
        messageState := fields[pdufield.MessageState]

        log.Printf("DLR recebido para MessageID=%s, State=%v", messageID, messageState)

        // Atualizar status no banco de dados
        updateMessageStatus(messageID.(string), messageState.(uint8))
    }
}

func updateMessageStatus(msgID string, state uint8) {
    // Salvar status no BD
    log.Printf("Atualizando status de %s para %d", msgID, state)
}
```

---

### 4. Enviar SMS em Massa

**Cenário**: Campanhas de marketing ou notificações em massa.

```go
package main

import (
    "log"
    "time"
    "golang.org/x/time/rate"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    // Configurar rate limiter (10 SMS/segundo)
    limiter := rate.NewLimiter(10, 5)

    tx := &smpp.Transmitter{
        Addr:        "smsc.example.com:2775",
        User:        "your_username",
        Passwd:      "your_password",
        RateLimiter: limiter,  // Aplicar rate limiting
        WindowSize:  100,      // Máx 100 requests concorrentes
    }

    conn := tx.Bind()
    <-conn

    log.Println("Conectado! Iniciando envio em massa...")

    // Lista de destinatários
    recipients := []string{
        "5511999999999",
        "5511888888888",
        "5511777777777",
        // ... milhares de números
    }

    // Enviar em paralelo (goroutines)
    sem := make(chan struct{}, 50)  // Limitar a 50 goroutines concorrentes

    for i, recipient := range recipients {
        sem <- struct{}{}  // Adquirir semaphore

        go func(i int, dst string) {
            defer func() { <-sem }()  // Liberar semaphore

            resp, err := tx.Submit(&smpp.ShortMessage{
                Src:  "Campaign",
                Dst:  dst,
                Text: pdutext.GSM7("Promoção especial para você!"),
            })

            if err != nil {
                log.Printf("[%d] Erro ao enviar para %s: %v", i, dst, err)
            } else {
                log.Printf("[%d] Enviado para %s - MsgID: %s", i, dst, resp.MessageID)
            }
        }(i, recipient)
    }

    // Aguardar todas goroutines finalizarem
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }

    log.Println("Envio em massa concluído!")
    tx.Close()
}
```

---

### 5. Mensagens Longas (SMS com mais de 160 caracteres)

**Cenário**: Enviar mensagens longas automaticamente divididas.

```go
package main

import (
    "log"
    "strings"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    tx := &smpp.Transmitter{
        Addr:   "smsc.example.com:2775",
        User:   "your_username",
        Passwd: "your_password",
    }

    conn := tx.Bind()
    <-conn

    // Mensagem longa (mais de 160 caracteres)
    longMessage := strings.Repeat("Esta é uma mensagem muito longa que será dividida automaticamente. ", 5)

    log.Printf("Enviando mensagem com %d caracteres...", len(longMessage))

    // SubmitLongMsg divide automaticamente
    resps, err := tx.SubmitLongMsg(&smpp.ShortMessage{
        Src:  "MyApp",
        Dst:  "5511999999999",
        Text: pdutext.GSM7(longMessage),
    })

    if err != nil {
        log.Fatal("Erro:", err)
    }

    log.Printf("Mensagem dividida em %d partes:", len(resps))
    for i, resp := range resps {
        log.Printf("  Parte %d - MessageID: %s", i+1, resp.MessageID)
    }

    tx.Close()
}
```

---

### 6. Monitoramento de Conexão

**Cenário**: Aplicação de longa duração que precisa monitorar status de conexão.

```go
package main

import (
    "log"
    "time"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    tx := &smpp.Transmitter{
        Addr:        "smsc.example.com:2775",
        User:        "your_username",
        Passwd:      "your_password",
        EnquireLink: 10 * time.Second,  // Keepalive a cada 10s
    }

    // Monitorar status de conexão
    conn := tx.Bind()

    go func() {
        for status := range conn {
            switch status.Status() {
            case smpp.Connected:
                log.Println("✓ Conectado ao SMSC")
            case smpp.Disconnected:
                log.Println("✗ Desconectado - Reconectando...")
            }

            if status.Error() != nil {
                log.Printf("Erro: %v", status.Error())
            }
        }
    }()

    // Aguardar primeira conexão
    <-time.After(1 * time.Second)

    // Loop de envio periódico
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        resp, err := tx.Submit(&smpp.ShortMessage{
            Src:  "Monitor",
            Dst:  "5511999999999",
            Text: pdutext.Raw("Heartbeat"),
        })

        if err != nil {
            log.Printf("Erro ao enviar: %v", err)
        } else {
            log.Printf("Enviado - MsgID: %s", resp.MessageID)
        }
    }
}
```

---

### 7. Integração com Banco de Dados

**Cenário**: Persistir mensagens enviadas e recebidas.

```go
package main

import (
    "database/sql"
    "log"
    "time"
    _ "github.com/lib/pq"  // PostgreSQL driver
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

var db *sql.DB

func main() {
    var err error
    db, err = sql.Open("postgres", "postgres://user:pass@localhost/sms_db?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Transceiver para enviar e receber
    tc := &smpp.Transceiver{
        Addr:    "smsc.example.com:2775",
        User:    "your_username",
        Passwd:  "your_password",
        Handler: handleIncomingPDU,
    }

    conn := tc.Bind()
    <-conn

    // Enviar SMS e salvar no BD
    sendAndSave(tc, "5511999999999", "Hello from DB integration!")

    // Manter rodando para receber
    select {}
}

func sendAndSave(tc *smpp.Transceiver, dst, message string) {
    resp, err := tc.Submit(&smpp.ShortMessage{
        Src:      "MyApp",
        Dst:      dst,
        Text:     pdutext.Raw(message),
        Register: smpp.FinalDeliveryReceipt,
    })

    if err != nil {
        log.Printf("Erro ao enviar: %v", err)
        return
    }

    // Salvar no banco de dados
    _, err = db.Exec(`
        INSERT INTO sms_outbound (message_id, destination, message, status, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `, resp.MessageID, dst, message, "SENT", time.Now())

    if err != nil {
        log.Printf("Erro ao salvar no BD: %v", err)
    } else {
        log.Printf("SMS enviado e salvo - MsgID: %s", resp.MessageID)
    }
}

func handleIncomingPDU(p pdu.Body) {
    if p.Header().ID != pdu.DeliverSMID {
        return
    }

    fields := p.Fields()
    esmClass := fields[pdufield.ESMClass].(uint8)

    // Delivery Receipt
    if esmClass&0x04 != 0 {
        messageID := fields[pdufield.ReceiptedMessageID].(string)
        messageState := fields[pdufield.MessageState].(uint8)

        // Atualizar status no BD
        _, err := db.Exec(`
            UPDATE sms_outbound SET status = $1, updated_at = $2
            WHERE message_id = $3
        `, stateToString(messageState), time.Now(), messageID)

        if err != nil {
            log.Printf("Erro ao atualizar BD: %v", err)
        } else {
            log.Printf("Status atualizado para MsgID=%s", messageID)
        }
    } else {
        // SMS recebido
        src := fields[pdufield.SourceAddr].(string)
        dst := fields[pdufield.DestinationAddr].(string)
        text := fields[pdufield.ShortMessage].([]byte)

        // Salvar SMS recebido
        _, err := db.Exec(`
            INSERT INTO sms_inbound (source, destination, message, received_at)
            VALUES ($1, $2, $3, $4)
        `, src, dst, string(text), time.Now())

        if err != nil {
            log.Printf("Erro ao salvar SMS recebido: %v", err)
        } else {
            log.Printf("SMS recebido de %s salvo no BD", src)
        }
    }
}

func stateToString(state uint8) string {
    states := map[uint8]string{
        0: "SCHEDULED",
        1: "ENROUTE",
        2: "DELIVERED",
        3: "EXPIRED",
        4: "DELETED",
        5: "UNDELIVERABLE",
        6: "ACCEPTED",
        7: "UNKNOWN",
        8: "REJECTED",
    }
    if s, ok := states[state]; ok {
        return s
    }
    return "UNKNOWN"
}
```

---

### 8. Integração com Fila de Mensagens (RabbitMQ)

**Cenário**: Desacoplar envio de SMS com fila de mensagens.

```go
package main

import (
    "encoding/json"
    "log"
    "github.com/streadway/amqp"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

type SMSMessage struct {
    Destination string `json:"destination"`
    Message     string `json:"message"`
}

func main() {
    // Conectar ao RabbitMQ
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Fatal(err)
    }
    defer ch.Close()

    // Declarar fila
    q, err := ch.QueueDeclare("sms_queue", true, false, false, false, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Consumir mensagens da fila
    msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Conectar ao SMSC
    tx := &smpp.Transmitter{
        Addr:   "smsc.example.com:2775",
        User:   "your_username",
        Passwd: "your_password",
    }

    connStatus := tx.Bind()
    <-connStatus

    log.Println("Worker iniciado, aguardando mensagens da fila...")

    // Processar mensagens da fila
    for msg := range msgs {
        var sms SMSMessage
        if err := json.Unmarshal(msg.Body, &sms); err != nil {
            log.Printf("Erro ao parsear JSON: %v", err)
            msg.Nack(false, false)  // Descartar mensagem inválida
            continue
        }

        // Enviar SMS
        resp, err := tx.Submit(&smpp.ShortMessage{
            Src:  "Worker",
            Dst:  sms.Destination,
            Text: pdutext.Raw(sms.Message),
        })

        if err != nil {
            log.Printf("Erro ao enviar SMS: %v", err)
            msg.Nack(false, true)  // Requeue para retry
        } else {
            log.Printf("SMS enviado - MsgID: %s", resp.MessageID)
            msg.Ack(false)  // Confirmar processamento
        }
    }
}
```

---

## Configuração de Produção

### Variáveis de Ambiente

**Recomendação**: Nunca hardcode credenciais. Use variáveis de ambiente.

```go
package main

import (
    "os"
    "time"
    "github.com/devyx-tech/go-smpp/smpp"
)

func main() {
    tx := &smpp.Transmitter{
        Addr:        os.Getenv("SMSC_ADDR"),           // smsc.example.com:2775
        User:        os.Getenv("SMSC_USER"),           // username
        Passwd:      os.Getenv("SMSC_PASSWORD"),       // password
        EnquireLink: 10 * time.Second,
        RespTimeout: 5 * time.Second,
    }

    // ...
}
```

**.env** file:
```bash
SMSC_ADDR=smsc.example.com:2775
SMSC_USER=your_username
SMSC_PASSWORD=your_password
```

---

### TLS/SSL

**Uso**: Para conexões seguras ao SMSC.

```go
import (
    "crypto/tls"
    "github.com/devyx-tech/go-smpp/smpp"
)

func main() {
    tx := &smpp.Transmitter{
        Addr:   "smsc.example.com:2776",  // Porta TLS (geralmente 2776)
        User:   "username",
        Passwd: "password",
        TLS: &tls.Config{
            InsecureSkipVerify: false,  // Validar certificado em produção
            ServerName:         "smsc.example.com",
        },
    }

    // ...
}
```

---

### Logging e Observabilidade

**Middleware de Logging**:

```go
package main

import (
    "log"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
)

type loggingConn struct {
    next smpp.Conn
}

func (c *loggingConn) Read() (pdu.Body, error) {
    p, err := c.next.Read()
    if err == nil {
        log.Printf("← PDU recebido: %s (seq=%d)", p.Header().ID, p.Header().Seq)
    }
    return p, err
}

func (c *loggingConn) Write(p pdu.Body) error {
    log.Printf("→ PDU enviado: %s (seq=%d)", p.Header().ID, p.Header().Seq)
    return c.next.Write(p)
}

func (c *loggingConn) Close() error {
    log.Println("Conexão fechada")
    return c.next.Close()
}

func loggingMiddleware(next smpp.Conn) smpp.Conn {
    return &loggingConn{next: next}
}

func main() {
    tx := &smpp.Transmitter{
        Addr:       "smsc.example.com:2775",
        User:       "username",
        Passwd:     "password",
        Middleware: loggingMiddleware,  // Aplicar middleware
    }

    // Todos os PDUs serão logados
    tx.Bind()
    // ...
}
```

---

### Métricas (Prometheus)

```go
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
)

var (
    smsSent = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "sms_sent_total",
            Help: "Total SMS enviados",
        },
        []string{"status"},
    )

    smsReceived = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "sms_received_total",
            Help: "Total SMS recebidos",
        },
    )
)

func init() {
    prometheus.MustRegister(smsSent)
    prometheus.MustRegister(smsReceived)
}

func main() {
    // Expor métricas HTTP
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":9090", nil)

    // Transceiver com tracking de métricas
    tc := &smpp.Transceiver{
        Addr:    "smsc.example.com:2775",
        User:    "username",
        Passwd:  "password",
        Handler: handleWithMetrics,
    }

    tc.Bind()

    // Enviar e trackear
    resp, err := tc.Submit(&smpp.ShortMessage{
        Src:  "Metrics",
        Dst:  "5511999999999",
        Text: pdutext.Raw("Test"),
    })

    if err != nil {
        smsSent.WithLabelValues("error").Inc()
    } else {
        smsSent.WithLabelValues("success").Inc()
    }

    select {}
}

func handleWithMetrics(p pdu.Body) {
    if p.Header().ID == pdu.DeliverSMID {
        smsReceived.Inc()
    }
}
```

---

## Melhores Práticas

### 1. Sempre Use Context para Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Use ctx em operações
```

### 2. Trate Erros de Forma Granular

```go
_, err := tx.Submit(sm)
if err != nil {
    switch {
    case err == smpp.ErrNotConnected:
        // Aguardar reconexão
    case err == smpp.ErrTimeout:
        // Retry
    default:
        if status, ok := err.(pdu.Status); ok {
            // Erro SMPP específico
            handleSMPPError(status)
        }
    }
}
```

### 3. Use Rate Limiting para Respeitar Limites do SMSC

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 5)  // 10 msgs/s, burst de 5

tx := &smpp.Transmitter{
    RateLimiter: limiter,
}
```

### 4. Monitorar Status de Conexão

```go
go func() {
    for status := range tx.Bind() {
        if status.Status() == smpp.Disconnected {
            // Pausar envios ou alertar
        }
    }
}()
```

### 5. Graceful Shutdown

```go
import "os/signal"

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

<-sigChan
log.Println("Desligando...")

// Fechar conexões gracefully
tx.Close()

// Aguardar inflight requests
time.Sleep(2 * time.Second)

os.Exit(0)
```

---

## Troubleshooting

### Problema: "not connected" errors

**Causa**: Conexão ainda não estabelecida ou caiu

**Solução**:
```go
conn := tx.Bind()
<-conn  // Aguardar primeira conexão

// Ou verificar status
if status.Status() != smpp.Connected {
    log.Println("Aguardando conexão...")
}
```

---

### Problema: "timeout waiting for response"

**Causa**: SMSC não respondeu em tempo ou rede lenta

**Solução**:
```go
tx := &smpp.Transmitter{
    RespTimeout: 10 * time.Second,  // Aumentar timeout
}
```

---

### Problema: SMS não entregue

**Causa**: Múltiplas possíveis (número inválido, saldo, bloqueio)

**Solução**:
1. Solicitar delivery receipt
2. Consultar com QuerySM
3. Verificar logs do SMSC

```go
sm := &smpp.ShortMessage{
    Register: smpp.FinalDeliveryReceipt,
}
```

---

### Problema: Mensagens duplicadas

**Causa**: Retry automático após timeout mas mensagem foi enviada

**Solução**: Implementar idempotência no lado do servidor ou usar MessageID para deduplicação.

---

## Próximos Passos

Consulte a documentação completa:
- [**Funcionalidades**](features.md) - Detalhes de cada feature
- [**Regras de Negócio**](business-rules.md) - Protocolo SMPP
- [**Stack**](stack.md) - Arquitetura
- [**Padrões**](patterns.md) - Design patterns

---

## Suporte

Para issues e contribuições:
- **GitHub**: https://github.com/devyx-tech/go-smpp
- **Issues**: https://github.com/devyx-tech/go-smpp/issues

---

## Recursos Adicionais

- [Especificação SMPP 3.4](http://www.smsforum.net/)
- [Documentação Go](https://golang.org/doc/)
- [Exemplo de aplicações](../cmd/)
