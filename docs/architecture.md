# Arquitetura

**Data de Análise:** 2026-03-24

## Padrão Geral

**Padrão:** Arquitetura em camadas com separação Protocol Data Unit (PDU) / Transport / Client API.

**Como é aplicado neste repositório:**
- A camada PDU (`smpp/pdu/`) é completamente independente — serializa e deserializa bytes do protocolo SMPP 3.4
- A camada Transport (`smpp/conn.go`) gerencia conexões TCP/TLS e lê/escreve PDUs
- A camada Client API (`smpp/transmitter.go`, `smpp/receiver.go`, `smpp/transceiver.go`) expõe structs de alto nível para consumidores
- A camada Server (`smpp/server.go`) fornece um servidor SMPP com handlers customizáveis

## Camadas

**PDU (Protocol Data Unit):**
- Propósito: encoding/decoding binário de mensagens SMPP 3.4
- Localização: `smpp/pdu/`
- Contém: `Body` interface, `Codec` struct (base de todos os PDUs), `Header`, factory functions (`NewSubmitSM`, `NewBindTransmitter`, etc.)
- Sub-pacotes:
  - `smpp/pdu/pdufield/` — tipos de campos PDU (Fixed, Variable, SM, UDH) e serialização
  - `smpp/pdu/pdutext/` — codecs de texto (GSM7, UCS2, Latin1, ISO-8859-5, Raw)
  - `smpp/pdu/pdutlv/` — campos TLV (Tag-Length-Value) opcionais do SMPP
- Depende de: nenhuma outra camada interna
- Usada por: Transport, Client API, Server

**Transport:**
- Propósito: gerenciar conexões TCP/TLS, ler/escrever PDUs binários
- Localização: `smpp/conn.go`
- Contém: interfaces `Conn`, `Reader`, `Writer`, `Closer`; structs `conn` (conexão básica) e `connSwitch` (conexão com hot-swap para reconexão)
- Depende de: PDU (para ler/escrever `pdu.Body`)
- Usada por: Client API, Server

**Client API:**
- Propósito: API de alto nível para enviar/receber SMS
- Localização: `smpp/client.go`, `smpp/transmitter.go`, `smpp/receiver.go`, `smpp/transceiver.go`
- Contém:
  - `client` — gerenciador de conexão persistente com reconexão automática e backoff
  - `Transmitter` — cliente SMPP para envio (SubmitSM, SubmitMulti, QuerySM)
  - `Receiver` — cliente SMPP para recepção (DeliverSM) com merge de mensagens longas
  - `Transceiver` — combina Transmitter e Receiver via composição
- Depende de: Transport, PDU
- Usada por: aplicações consumidoras

**Server:**
- Propósito: servidor SMPP com autenticação e handlers customizáveis
- Localização: `smpp/server.go`
- Contém: interfaces `Server`, `Session`; handlers `RequestHandlerFunc`, `AuthRequestHandlerFunc`
- Depende de: Transport, PDU
- Usada por: aplicações que hospedam um servidor SMPP

**Encoding:**
- Propósito: implementação do codec GSM 7-bit com suporte a tabela estendida
- Localização: `smpp/encoding/gsm7.go`
- Contém: encoder/decoder GSM7 implementando `golang.org/x/text/encoding.Encoding`
- Depende de: `golang.org/x/text`
- Usada por: `smpp/pdu/pdutext/gsm7.go`

## Fluxo de Dados

**Envio de SMS (Transmitter.Submit):**

1. Consumidor cria `ShortMessage` com `Src`, `Dst`, `Text` (codec)
2. `Transmitter.Submit()` em `smpp/transmitter.go:317` cria PDU via `pdu.NewSubmitSM()`
3. `submitMsg()` em `smpp/transmitter.go:395` popula campos do PDU via `pdufield.Map.Set()`
4. `do()` em `smpp/transmitter.go:276` envia PDU via `client.Write()` e aguarda resposta no channel `inflight`
5. `client.Write()` em `smpp/client.go:244` aplica rate limiting e delega para `connSwitch.Write()`
6. `conn.Write()` em `smpp/conn.go:99` serializa PDU para bytes e escreve no TCP via `bufio.Writer`
7. Resposta chega via `client.Bind()` loop em `smpp/client.go:157`, é roteada para `handlePDU()`
8. `handlePDU()` em `smpp/transmitter.go:118` encontra o channel `inflight[seq]` e entrega a resposta

**Recepção de SMS (Receiver):**

1. `Receiver.Bind()` em `smpp/receiver.go:72` cria `client` e inicia loop de conexão
2. Após bind, `handlePDU()` em `smpp/receiver.go:149` lê PDUs do servidor
3. Para cada DeliverSM recebido, envia `DeliverSMResp` automático (se não em `SkipAutoRespondIDs`)
4. Se `MergeInterval > 0`, acumula partes de mensagens longas via UDH em `mergeHolders`
5. Quando todas as partes chegam, junta os buffers e chama `Handler(p)` com o PDU completo

**Conexão persistente (client.Bind):**

1. `client.Bind()` em `smpp/client.go:132` faz loop infinito até `Close()`
2. `Dial()` em `smpp/conn.go:58` cria conexão TCP (opcionalmente TLS)
3. `BindFunc` executa bind específico (Transmitter/Receiver/Transceiver)
4. `enquireLink()` em `smpp/client.go:192` envia heartbeats periódicos
5. Se conexão cai, backoff exponencial (delay *= e, max 120s) ou `BindInterval` fixo

## Abstrações Chave

**`pdu.Body` interface:**
- Propósito: abstração para qualquer PDU SMPP
- Localização: `smpp/pdu/body.go`
- Métodos: `Header()`, `Len()`, `FieldList()`, `Fields()`, `TLVFields()`, `SerializeTo()`
- Usado por: todo o codebase — é o tipo fundamental trocado em leituras e escritas

**`pdu.Codec` struct:**
- Propósito: implementação base de `pdu.Body` — todos os tipos de PDU embeddam `Codec`
- Localização: `smpp/pdu/codec.go`
- Padrão: cada tipo de PDU (SubmitSM, BindTransmitter, etc.) é um struct que embeds `*Codec`

**`connSwitch` struct:**
- Propósito: wrapper thread-safe de `Conn` que permite trocar a conexão subjacente durante reconexão
- Localização: `smpp/conn.go:122`
- Usado por: `client`, `session`

**`pdutext.Codec` interface:**
- Propósito: abstração para codificação de texto SMS
- Localização: `smpp/pdu/pdutext/codec.go`
- Implementações: `Raw`, `GSM7`, `UCS2`, `Latin1`, `ISO88595`

## Entry Points

**Biblioteca (consumo como dependência):**
- Import: `github.com/devyx-tech/go-smpp/smpp`
- Tipos principais: `Transmitter`, `Receiver`, `Transceiver`, `Server`
- Trigger: chamada a `Bind()` inicia conexão

**CLI:**
- Localização: `cmd/sms/main.go`
- Trigger: `go run ./cmd/sms/ send|query`
- Responsabilidades: parsing de flags, criação de `Transmitter`, envio/consulta de SMS

## Tratamento de Erros

**Estratégia:** erros sentinela para estados de conexão, `pdu.Status` como tipo de erro para respostas SMPP.

```go
// De smpp/conn.go
var (
    ErrNotConnected = errors.New("not connected")
    ErrNotBound     = errors.New("not bound")
    ErrTimeout      = errors.New("timeout waiting for response")
)
```

```go
// De smpp/transmitter.go — erros de resposta SMPP retornados como pdu.Status
if s := resp.PDU.Header().Status; s != 0 {
    return sm, s
}
```

## Concerns Transversais

**Logging:** `log` (stdlib) — formato texto — destino stdout. Usado apenas no server e smpptest, não na biblioteca client.
**Validação:** mínima — limite de 254 destinos em SubmitMulti (`smpp/transmitter.go:450`), validação de bind response ID. Maioria da validação é delegada ao SMSC.
**Concorrência:** `sync.Mutex` e channels para thread-safety. `connSwitch` protege read/write. `inflight` map protegido por mutex para correlação request/response.

---

*Análise de arquitetura: 2026-03-24*
