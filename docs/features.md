# Funcionalidades

**Data de Análise:** 2026-03-24

## Funcionalidades Principais

### Envio de SMS (Transmitter)

**O que faz:** Envia mensagens SMS para um SMSC via binding do tipo Transmitter do protocolo SMPP 3.4.
**Casos de uso:** Notificações, marketing SMS, OTPs, alertas.
**Componentes envolvidos:**
- `smpp/transmitter.go` — struct `Transmitter`, métodos `Submit()`, `SubmitLongMsg()`, `QuerySM()`
- `smpp/client.go` — gerenciamento de conexão persistente com reconnect
- `smpp/pdu/types.go` — PDUs `SubmitSM`, `SubmitSMResp`, `SubmitMulti`, `SubmitMultiResp`
**Dependências:** Conexão TCP/TLS com SMSC

**Operações:**
- `Submit()` — envia mensagem simples (SubmitSM) ou multi-destinatário (SubmitMulti)
- `SubmitLongMsg()` — fragmenta mensagem longa (>140 bytes) com UDH e envia partes
- `QuerySM()` — consulta status de entrega de uma mensagem enviada

### Recepção de SMS (Receiver)

**O que faz:** Recebe mensagens (DeliverSM) de um SMSC via binding do tipo Receiver.
**Casos de uso:** Recepção de respostas, delivery receipts, SMS bidirecional.
**Componentes envolvidos:**
- `smpp/receiver.go` — struct `Receiver`, handler function, merge de mensagens longas
**Dependências:** Conexão TCP/TLS com SMSC

**Características:**
- Handler function customizável (`HandlerFunc`)
- Auto-resposta de DeliverSMResp (configurável via `SkipAutoRespondIDs`)
- Merge automático de mensagens longas concatenadas via UDH
- Cleanup periódico de partes expiradas

### Transceiver (Envio + Recepção)

**O que faz:** Combina Transmitter e Receiver em uma única conexão SMPP.
**Casos de uso:** Aplicações que enviam e recebem SMS na mesma conexão.
**Componentes envolvidos:**
- `smpp/transceiver.go` — struct `Transceiver` (embute `Transmitter`)
**Dependências:** Conexão TCP/TLS com SMSC

### Servidor SMPP

**O que faz:** Servidor SMPP para aplicações que atuam como SMSC ou gateway.
**Casos de uso:** Gateways SMS, agregadores, testes de integração.
**Componentes envolvidos:**
- `smpp/server.go` — interfaces `Server` e `Session`, handlers customizáveis
**Dependências:** `net.Listener` (TCP)

**Características:**
- Autenticação via Bind PDU (customizável com `HandleAuth`)
- Handlers por tipo de PDU via `Handle(id, func)`
- Gerenciamento de sessions com IDs aleatórios
- Suporte a TLS

### Codificação de Texto

**O que faz:** Codecs de texto para converter mensagens entre formatos suportados pelo SMPP.
**Componentes envolvidos:**
- `smpp/pdu/pdutext/` — codecs: `GSM7`, `Latin1`, `UCS2`, `ISO88595`, `Raw`
- `smpp/encoding/gsm7.go` — encoder/decoder GSM 7-bit (packed e unpacked)

| Codec | DataCoding | Descrição |
|---|---|---|
| `GSM7` | 0x00 | GSM 7-bit default alphabet |
| `Latin1` | 0x03 | Windows-1252 / CP1252 |
| `ISO88595` | 0x06 | ISO-8859-5 (Cyrillic) |
| `UCS2` | 0x08 | UTF-16-BE (Unicode) |
| `Raw` | 0x00 | Sem codificação (passthrough) |

### Codificação/Decodificação de PDUs

**O que faz:** Serialização e deserialização binária de PDUs SMPP 3.4.
**Componentes envolvidos:**
- `smpp/pdu/codec.go` — `Codec` base com `SerializeTo()` e `Decode()`
- `smpp/pdu/header.go` — Header 16 bytes, `Status` com 30+ códigos de erro
- `smpp/pdu/types.go` — 19 tipos de PDU definidos
- `smpp/pdu/pdufield/` — campos obrigatórios
- `smpp/pdu/pdutlv/` — campos TLV opcionais (50+ tags)

## Funcionalidades Secundárias

- **Conexão Persistente:** Reconnect automático com backoff exponencial (fator `e`, max 120s) ou intervalo fixo (`BindInterval`) — `smpp/client.go:132`
- **EnquireLink (Keepalive):** Heartbeat periódico (mín. 10s) com timeout configurável. Unbind automático se SMSC não responder — `smpp/client.go:192`
- **Rate Limiting:** Interface `RateLimiter` compatível com `golang.org/x/time/rate` — `smpp/client.go:84`
- **Window Size:** Limite de mensagens inflight. Retorna `ErrMaxWindowSize` — `smpp/transmitter.go:276`
- **Submit Multi:** Envio para até 254 destinatários + listas de distribuição — `smpp/transmitter.go:448`
- **TLV Fields:** 50+ tags TLV padrão SMPP para campos opcionais — `smpp/pdu/pdutlv/`
- **JSON Serialization:** `Codec.MarshalJSON()` e `UnmarshalJSON()` para debug — `smpp/pdu/codec.go:110`
- **ConnMiddleware:** Interceptor de conexão para logging/métricas — `smpp/client.go:74`
- **Servidor de Teste:** `smpptest.Server` com echo handler e broadcast — `smpp/smpptest/`
- **CLI (smppcli):** Envio (`send`) e consulta (`query`) de SMS via linha de comando — `cmd/sms/main.go`

## PDUs Não Implementados

- AlertNotification, CancelSM/CancelSMResp, ReplaceSM/ReplaceSMResp, DataSM/DataSMResp, Outbind

---

*Análise de funcionalidades: 2026-03-24*
