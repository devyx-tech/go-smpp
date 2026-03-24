# Integrações

**Data de Análise:** 2026-03-24

## Integração Primária: SMSC

### SMSC (Short Message Service Center)

- **Tipo:** Protocolo binário SMPP 3.4 sobre TCP
- **Propósito:** Enviar e receber SMS. O SMSC é o gateway entre a aplicação e a rede móvel.
- **Protocolo:** SMPP 3.4 (binário, big-endian, sobre TCP)
- **Porta padrão:** 2775 (`smpp/conn.go:61` — default do `Dial()`)
- **Dados trocados:**
  - **Enviados:** BindTransmitter/Receiver/Transceiver, SubmitSM, SubmitMulti, QuerySM, EnquireLink, Unbind
  - **Recebidos:** BindResp, SubmitSMResp, SubmitMultiResp, QuerySMResp, DeliverSM, EnquireLinkResp, UnbindResp
- **Dependência:** Crítica — sem SMSC não há envio/recebimento de SMS
- **Tratamento de falhas:**
  - Reconnect automático com backoff exponencial (fator `e`, max 120s)
  - EnquireLink como heartbeat com timeout configurável
  - Window size para controlar mensagens inflight
  - Rate limiting para respeitar limites do SMSC

### Segurança da Conexão

- **TLS opcional:** Configurável via campo `TLS *tls.Config` nos clients/server
- **Autenticação:** Via campos `system_id` e `password` no bind PDU
- **Credenciais no CLI:** Via env vars (`SMPP_USER`, `SMPP_PASSWD`) ou flags `--user`/`--passwd`

## Dependências Externas (Bibliotecas)

### golang.org/x/text

- **Tipo:** Biblioteca Go
- **Propósito:** Encoding/decoding de texto em character sets ISO
- **Uso:** Conversão para Latin1 (Windows-1252), UCS2 (UTF-16-BE), ISO-8859-5
- **Dependência:** Crítica para codificação correta de mensagens
- **Fallback:** Sem fallback — texto não-ASCII fica corrompido sem esta biblioteca

### golang.org/x/time

- **Tipo:** Biblioteca Go
- **Propósito:** Rate limiting com token bucket algorithm
- **Uso:** Interface `RateLimiter` compatível com `rate.Limiter`
- **Dependência:** Opcional — usado apenas se rate limiting é configurado
- **Fallback:** Sem rate limiter, mensagens são enviadas sem controle de taxa

### golang.org/x/net

- **Tipo:** Biblioteca Go
- **Propósito:** Pacote `context` para suporte a cancelamento no rate limiter
- **Dependência:** Indireta — necessária pelo rate limiter

### github.com/urfave/cli

- **Tipo:** Biblioteca Go
- **Propósito:** Framework CLI para `smppcli`
- **Uso:** Apenas em `cmd/sms/main.go` — não afeta a biblioteca principal
- **Dependência:** Opcional — somente para o CLI tool

## Protocolo Wire Format

```
[4 bytes: Command Length (big-endian)]
[4 bytes: Command ID (big-endian)]
[4 bytes: Command Status (big-endian)]
[4 bytes: Sequence Number (big-endian)]
[N bytes: Body (mandatory fields + optional TLV fields)]
```

## DataCoding Suportados

| Valor | Tipo | Charset |
|---|---|---|
| 0x00 | GSM 7-bit | SMSC Default Alphabet |
| 0x03 | Latin1 | Windows-1252 |
| 0x06 | ISO-8859-5 | Cyrillic |
| 0x08 | UCS2 | UTF-16-BE |

## Resiliência

### Reconnect com Backoff Exponencial
- Delay inicial: 1 segundo
- Fator de crescimento: `e` (~2.718)
- Delay máximo: 120 segundos
- Alternativa: `BindInterval` fixo (intervalo constante)
- **Implementação:** `smpp/client.go:133-188`

### EnquireLink Heartbeat
- Intervalo configurável (mínimo 10s)
- Timeout: 3x o intervalo (padrão) ou customizado via `EnquireLinkTimeout`
- Ação no timeout: Unbind + Close + reconnect
- **Implementação:** `smpp/client.go:192-218`

### Window Size
- Limita mensagens inflight para evitar sobrecarga no SMSC
- Erro `ErrMaxWindowSize` quando limite excedido
- **Implementação:** `smpp/transmitter.go:283-289`

### Rate Limiting
- Interface `RateLimiter` com método `Wait(ctx)`
- Compatível com `golang.org/x/time/rate.Limiter`
- Aplicado antes de cada envio de PDU
- **Implementação:** `smpp/client.go:245-248`

### Cleanup de Mensagens Parciais
- Partes de mensagens longas no Receiver expiram após `MergeInterval`
- Goroutine de cleanup periódico (`MergeCleanupInterval`, default 1s)
- Evita vazamento de memória com partes órfãs
- **Implementação:** `smpp/receiver.go:246-264`

---

*Análise de integrações: 2026-03-24*
