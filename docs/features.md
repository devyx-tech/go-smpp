# Funcionalidades

## Funcionalidades Principais

### 1. Envio de SMS (Transmitter)
- **Descricao**: Envia mensagens curtas (SMS) para um SMSC via protocolo SMPP, usando binding do tipo Transmitter.
- **Casos de Uso**: Sistemas de notificacao, marketing SMS, OTPs, alertas.
- **Componentes Envolvidos**:
  - `smpp/transmitter.go` — Struct `Transmitter`, metodos `Submit()`, `SubmitLongMsg()`, `QuerySM()`
  - `smpp/client.go` — Gerenciamento de conexao persistente
  - `smpp/pdu/types.go` — PDUs: `SubmitSM`, `SubmitSMResp`, `SubmitMulti`, `SubmitMultiResp`
- **Dependencias**: Conexao TCP/TLS com SMSC

**Operacoes suportadas**:
- `Submit()` — Envia mensagem curta simples (SubmitSM) ou multi-destinatario (SubmitMulti)
- `SubmitLongMsg()` — Envia mensagem longa fragmentada com UDH (User Data Header)
- `QuerySM()` — Consulta status de entrega de uma mensagem

### 2. Recebimento de SMS (Receiver)
- **Descricao**: Recebe mensagens (DeliverSM) de um SMSC via binding do tipo Receiver.
- **Casos de Uso**: Recebimento de respostas de usuarios, delivery receipts, SMS bidirecional.
- **Componentes Envolvidos**:
  - `smpp/receiver.go` — Struct `Receiver`, handler function, merge de mensagens longas
- **Dependencias**: Conexao TCP/TLS com SMSC

**Caracteristicas**:
- Handler function customizavel (`HandlerFunc`)
- Auto-resposta de DeliverSMResp (configuravel via `SkipAutoRespondIDs`)
- Merge automatico de mensagens longas concatenadas (UDH)
- Cleanup periodico de partes de mensagens expiradas

### 3. Transceiver (Envio + Recebimento)
- **Descricao**: Combina as funcionalidades de Transmitter e Receiver em uma unica conexao.
- **Casos de Uso**: Aplicacoes que precisam enviar e receber SMS na mesma conexao.
- **Componentes Envolvidos**:
  - `smpp/transceiver.go` — Struct `Transceiver` (embute `Transmitter`)
- **Dependencias**: Conexao TCP/TLS com SMSC

### 4. Servidor SMPP (Producao)
- **Descricao**: Servidor SMPP para aplicacoes que atuam como SMSC ou gateway de mensagens.
- **Casos de Uso**: Gateways SMS, agregadores, testes de integracao end-to-end.
- **Componentes Envolvidos**:
  - `smpp/server.go` — Struct `server`, interfaces `Server` e `Session`
- **Dependencias**: `net.Listener` (TCP)

**Caracteristicas**:
- Autenticacao via Bind PDU (configuravel com `HandleAuth`)
- Handlers customizaveis por tipo de PDU (`Handle`)
- Gerenciamento de sessoes com IDs aleatorios
- Suporte a TLS

### 5. Codificacao de Texto
- **Descricao**: Codecs de texto para converter mensagens entre formatos.
- **Casos de Uso**: Envio de SMS em diferentes idiomas e character sets.
- **Componentes Envolvidos**:
  - `smpp/pdu/pdutext/` — Codecs: `GSM7`, `Latin1`, `UCS2`, `ISO88595`, `Raw`
  - `smpp/encoding/gsm7.go` — Encoder/decoder GSM 7-bit (packed e unpacked)

**Codecs suportados**:
| Codec | DataCoding | Descricao |
|---|---|---|
| `GSM7` | 0x00 | GSM 7-bit default alphabet (SMSC Default) |
| `Latin1` | 0x03 | Windows-1252 / CP1252 |
| `ISO88595` | 0x06 | ISO-8859-5 (Cyrillic) |
| `UCS2` | 0x08 | UTF-16-BE (Unicode) |
| `Raw` | 0x00 | Sem codificacao (passthrough) |

### 6. Codificacao/Decodificacao de PDUs
- **Descricao**: Serializacao e deserializacao binaria de PDUs SMPP conforme a especificacao 3.4.
- **Casos de Uso**: Toda comunicacao SMPP passa por esta camada.
- **Componentes Envolvidos**:
  - `smpp/pdu/codec.go` — `Codec` base com `SerializeTo()` e `Decode()`
  - `smpp/pdu/header.go` — Header de 16 bytes, `Status` com codigos de erro SMPP
  - `smpp/pdu/types.go` — Definicao de 19 tipos de PDU
  - `smpp/pdu/pdufield/` — Campos obrigatorios
  - `smpp/pdu/pdutlv/` — Campos opcionais TLV

**PDUs implementados**:
- Bind: BindReceiver, BindTransmitter, BindTransceiver (+ respostas)
- Mensagem: SubmitSM, DeliverSM, SubmitMulti (+ respostas)
- Consulta: QuerySM (+ resposta)
- Sessao: Unbind, EnquireLink, GenericNACK (+ respostas)

**PDUs nao implementados** (TODOs no codigo):
- AlertNotification, CancelSM, ReplaceSM, DataSM, Outbind

## Funcionalidades Secundarias

### Conexao Persistente com Reconnect
- Reconnect automatico com backoff exponencial (fator `e`, maximo 120s)
- Intervalo de bind fixo opcional (`BindInterval`)
- Notificacao de status via channel (`ConnStatus`)
- **Componente**: `smpp/client.go:132` — metodo `Bind()`

### EnquireLink (Keepalive)
- Envio periodico de EnquireLink PDU (intervalo configuravel, minimo 10s)
- Timeout configuravel para resposta (`EnquireLinkTimeout`, default 3x intervalo)
- Unbind automatico se o SMSC nao responder dentro do timeout
- **Componente**: `smpp/client.go:192` — metodo `enquireLink()`

### Rate Limiting
- Interface `RateLimiter` compativel com `golang.org/x/time/rate`
- Aplicado antes de cada Write no client
- **Componente**: `smpp/client.go:84`

### Window Size Control
- Limite de mensagens em voo (inflight)
- Retorna `ErrMaxWindowSize` quando o limite e excedido
- **Componente**: `smpp/transmitter.go:276-289`

### Mensagens Longas (UDH)
- Fragmentacao automatica em partes de 134 bytes (140 - 6 bytes UDH)
- Referencia de 8 bits para identificacao de partes
- **Envio**: `smpp/transmitter.go:333` — `SubmitLongMsg()`
- **Recebimento**: `smpp/receiver.go:149` — `handlePDU()` com merge

### Submit Multi
- Envio para multiplos destinatarios em um unico PDU
- Suporte a listas de distribuicao
- Maximo 254 destinatarios (`MaxDestinationAddress`)
- Informacao de destinatarios com falha via `UnsuccessSmes()`
- **Componente**: `smpp/transmitter.go:448`

### TLV (Tag-Length-Value)
- Suporte a 50+ tags TLV padrao SMPP
- Campos opcionais em PDUs via `TLVFields`
- Tipos: `String`, `CString`, `MessageStateType`
- **Componente**: `smpp/pdu/pdutlv/`

### Servidor de Teste (smpptest)
- Servidor SMPP leve para testes unitarios e de integracao
- Echo handler padrao (ecoa PDUs recebidos)
- Broadcast de mensagens para todos os clientes conectados
- **Componente**: `smpp/smpptest/`

### Serializacao JSON de PDUs
- `Codec.MarshalJSON()` e `UnmarshalJSON()` para serializacao de PDUs em JSON
- `pdufield.Map` serializa campos com representacao hex e texto
- Util para logging e debugging
- **Componente**: `smpp/pdu/codec.go:110-141`

### CLI (smppcli)
- Ferramenta de linha de comando para envio e consulta de SMS
- Comandos: `send` (enviar SMS), `query` (consultar status)
- Suporte a TLS, credenciais via flags ou env vars (`SMPP_USER`, `SMPP_PASSWD`)
- Codificacoes: raw, ucs2, latin1
- **Componente**: `cmd/sms/main.go`

## Funcionalidades em Desenvolvimento

Baseado nos TODOs encontrados no codigo:
- AlertNotification PDU (`pdu/codec.go`)
- CancelSM / CancelSMResp PDU (`pdu/codec.go`)
- ReplaceSM / ReplaceSMResp PDU (`pdu/codec.go`)
- DataSM / DataSMResp PDU (`pdu/codec.go`)
- Outbind PDU (`pdu/codec.go`)
- Validacao do UnbindResp no Close (`client.go:257`)
- Correcao do codec UCS2/Latin1 (`pdutext/doc.go`)
