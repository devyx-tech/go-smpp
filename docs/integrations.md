# Integracoes

## Visao Geral

O `go-smpp` e uma biblioteca cliente/servidor SMPP. Sua integracao primaria e com SMSCs (Short Message Service Centers) via protocolo SMPP 3.4 sobre TCP/TLS.

## Integracoes Primarias

### SMSC (Short Message Service Center)
- **Tipo**: Protocolo binario SMPP 3.4 sobre TCP
- **Proposito**: Enviar e receber mensagens SMS. O SMSC e o gateway entre a aplicacao e a rede movel.
- **Protocolo**: SMPP 3.4 (binario, big-endian, sobre TCP)
- **Porta padrao**: 2775
- **Dados Trocados**:
  - **Enviados**: BindTransmitter/Receiver/Transceiver, SubmitSM, SubmitMulti, QuerySM, EnquireLink, Unbind
  - **Recebidos**: BindResp, SubmitSMResp, SubmitMultiResp, QuerySMResp, DeliverSM, EnquireLinkResp, UnbindResp
- **Dependencia**: Critica — sem SMSC nao ha envio/recebimento de SMS
- **Tratamento de Falhas**:
  - Reconnect automatico com backoff exponencial (fator `e`, max 120s)
  - EnquireLink como heartbeat com timeout configuravel
  - Window size para controlar mensagens em voo
  - Rate limiting para respeitar limites do SMSC

### Seguranca da Conexao
- **TLS opcional**: Configuravel via campo `TLS *tls.Config` nos clients/server
- **Autenticacao**: Via campos `system_id` e `password` no bind PDU
- **Credenciais no CLI**: Suporte a variavel de ambiente (`SMPP_USER`, `SMPP_PASSWD`) ou flags

## Dependencias Externas (Bibliotecas)

### golang.org/x/text
- **Tipo**: Biblioteca Go
- **Proposito**: Codificacao e decodificacao de texto em diversos character sets
- **Uso**: Conversao de texto para Latin1 (Windows-1252), UCS2 (UTF-16-BE), ISO-8859-5
- **Dependencia**: Critica para codificacao correta de mensagens

### golang.org/x/time
- **Tipo**: Biblioteca Go
- **Proposito**: Rate limiting com token bucket algorithm
- **Uso**: Interface `RateLimiter` compativel com `rate.Limiter`
- **Dependencia**: Opcional — usado apenas se rate limiting e configurado

### golang.org/x/net
- **Tipo**: Biblioteca Go
- **Proposito**: Pacote `context` para suporte a cancelamento no rate limiter
- **Dependencia**: Indireta — necessaria apenas pelo rate limiter

### github.com/urfave/cli
- **Tipo**: Biblioteca Go
- **Proposito**: Framework de CLI para a ferramenta `smppcli`
- **Uso**: Apenas no `cmd/sms/main.go`, nao afeta a biblioteca principal
- **Dependencia**: Opcional — somente para o CLI tool

## Contratos de Integracao

### Protocolo SMPP 3.4
A biblioteca implementa o subconjunto mais comum do protocolo SMPP 3.4:

**PDUs suportados (request + response)**:
- `BindTransmitter` / `BindTransmitterResp`
- `BindReceiver` / `BindReceiverResp`
- `BindTransceiver` / `BindTransceiverResp`
- `SubmitSM` / `SubmitSMResp`
- `SubmitMulti` / `SubmitMultiResp`
- `DeliverSM` / `DeliverSMResp`
- `QuerySM` / `QuerySMResp`
- `EnquireLink` / `EnquireLinkResp`
- `Unbind` / `UnbindResp`
- `GenericNACK`

**Formato do wire protocol**:
```
[4 bytes: Command Length (big-endian)]
[4 bytes: Command ID (big-endian)]
[4 bytes: Command Status (big-endian)]
[4 bytes: Sequence Number (big-endian)]
[N bytes: Body (mandatory fields + optional TLV fields)]
```

### DataCoding suportados
| Valor | Tipo | Charset |
|---|---|---|
| 0x00 | GSM 7-bit | SMSC Default Alphabet |
| 0x03 | Latin1 | Windows-1252 |
| 0x06 | ISO-8859-5 | Cyrillic |
| 0x08 | UCS2 | UTF-16-BE |

## Resiliencia

### Reconnect com Backoff Exponencial
- Delay inicial: 1 segundo
- Fator de crescimento: `e` (~2.718)
- Delay maximo: 120 segundos
- Alternativa: `BindInterval` fixo
- **Implementacao**: `smpp/client.go:133-188`

### EnquireLink Heartbeat
- Intervalo configuravel (minimo 10s)
- Timeout: 3x o intervalo (padrao) ou customizado
- Acao no timeout: Unbind + Close + reconnect
- **Implementacao**: `smpp/client.go:192-218`

### Window Size
- Limita mensagens em voo para evitar sobrecarga no SMSC
- Erro `ErrMaxWindowSize` quando limite excedido
- **Implementacao**: `smpp/transmitter.go:283-289`

### Rate Limiting
- Interface `RateLimiter` com metodo `Wait(ctx)`
- Compativel com `golang.org/x/time/rate.Limiter`
- Aplicado antes de cada envio de PDU
- **Implementacao**: `smpp/client.go:245-248`

### Cleanup de Mensagens Parciais
- No Receiver, partes de mensagens longas tem expiracao configuravel (`MergeInterval`)
- Goroutine de cleanup periodico (`MergeCleanupInterval`, default 1s)
- Evita vazamento de memoria com partes orfas
- **Implementacao**: `smpp/receiver.go:246-264`
