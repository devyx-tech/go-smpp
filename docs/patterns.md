# Padroes de Design

## Padroes Arquiteturais

### Arquitetura em Camadas
O projeto segue uma separacao clara em tres camadas:

1. **Camada de Transporte** (`smpp/conn.go`): Abstrai conexoes TCP/TLS via interfaces `Conn`, `Reader`, `Writer`, `Closer`.
2. **Camada de Protocolo** (`smpp/pdu/`): Codecs binarios para PDUs SMPP — serializa e deserializa structs Go de/para bytes conforme a especificacao SMPP 3.4.
3. **Camada de Aplicacao** (`smpp/transmitter.go`, `receiver.go`, `transceiver.go`, `server.go`): Abstraccoes de alto nivel com gerenciamento de conexao, envio/recebimento de mensagens e handlers.

### Interface-Driven Design
Toda comunicacao entre camadas e feita via interfaces Go:

- `Conn` (`conn.go:32`) — combinacao de `Reader`, `Writer`, `Closer`
- `pdu.Body` (`pdu/body.go:18`) — interface abstrata para PDUs
- `pdutext.Codec` (`pdu/pdutext/codec.go:28`) — interface para codecs de texto
- `pdufield.Body` (`pdu/pdufield/body.go:10`) — interface para campos de PDU
- `ClientConn` (`client.go:59`) — interface para conexoes persistentes de cliente
- `Session` (`server.go:74`) — interface para sessoes do servidor
- `RateLimiter` (`client.go:84`) — interface para controle de taxa

## Padroes de Codigo

### Composition via Embedding
O `Transceiver` (`transceiver.go:36`) embute `Transmitter` para reutilizar toda a logica de envio (Submit, SubmitLongMsg, QuerySM) sem duplicacao:

```go
type Transceiver struct {
    // ... campos proprios
    Transmitter  // embedded
}
```

### Factory Pattern
`pdu.Factory` (`pdu/factory.go:16`) encapsula a criacao de PDUs com gerenciamento isolado de sequence numbers:

```go
type Factory interface {
    CreatePDU(id ID) (Body, error)
    CreatePDUResp(id ID, seq uint32) (Body, error)
}
```

Tambem ha constructors diretos (`pdu.NewSubmitSM()`, `pdu.NewBindTransmitter()`, etc.) para uso simples.

### Strategy Pattern (Codecs de Texto)
Os codecs de texto (`pdutext.Codec`) implementam o padrao Strategy — cada codec (`GSM7`, `Latin1`, `UCS2`, `ISO88595`, `Raw`) encapsula sua propria logica de encode/decode. A selecao do codec e feita pelo chamador ao configurar `ShortMessage.Text`.

### Observer Pattern (Connection Status)
O metodo `Bind()` retorna um channel `<-chan ConnStatus` que notifica mudancas de estado da conexao (Connected, Disconnected, ConnectionFailed, BindFailed). Consumidores observam o channel para reagir a mudancas:

```go
conn := tx.Bind()
for c := range conn {
    log.Println("Status:", c.Status())
}
```

### Decorator Pattern (ConnMiddleware)
`ConnMiddleware` (`client.go:74`) permite interceptar a conexao para logging, metricas ou monitoramento:

```go
type ConnMiddleware func(conn Conn) Conn
```

### Connection Switch (Hot-Swap)
`connSwitch` (`conn.go:122`) implementa `Conn` mas permite trocar a conexao subjacente de forma thread-safe, essencial para o mecanismo de reconexao transparente.

## Organizacao de Codigo

| Diretorio | Responsabilidade |
|---|---|
| `smpp/` | Pacote raiz — client, conn, transmitter, receiver, transceiver, server |
| `smpp/pdu/` | Codecs e tipos de PDU (Header, Body, Codec) |
| `smpp/pdu/pdufield/` | Campos obrigatorios de PDU (Fixed, Variable, SM, UDH) |
| `smpp/pdu/pdutext/` | Codecs de texto (GSM7, Latin1, UCS2, ISO-8859-5) |
| `smpp/pdu/pdutlv/` | Campos opcionais TLV (Tag-Length-Value) |
| `smpp/encoding/` | Codificacao GSM 7-bit (packed/unpacked) |
| `smpp/smpptest/` | Servidor SMPP de teste com echo handler |
| `cmd/sms/` | CLI `smppcli` — cliente de linha de comando |

## Convencoes de Nomenclatura

- **Pacotes**: lowercase, nomes curtos (`pdu`, `pdufield`, `pdutext`, `pdutlv`)
- **Interfaces**: nomes substantivos sem prefixo `I` (`Conn`, `Body`, `Codec`, `Session`)
- **Constructors**: `New` + tipo (`NewSubmitSM()`, `NewBindTransmitter()`)
- **Constructors com seq**: sufixo `Seq` (`NewDeliverSMRespSeq(seq)`)
- **Constantes de ID**: sufixo `ID` (`SubmitSMID`, `BindTransmitterRespID`)
- **Campos PDU**: nomes SMPP em snake_case como `Name` string (`source_addr`, `data_coding`)

## Padroes de Teste

- Framework: `testing` padrao Go
- Testes de unidade em arquivos `*_test.go` no mesmo pacote
- Testes de integracao usando `smpptest.Server` como mock SMPP server
- Pattern: criar servidor de teste, conectar client, exercitar operacoes
- Exemplos documentados em `example_test.go`

Arquivos de teste existentes:
- `smpp/conn_test.go`, `smpp/transmitter_test.go`, `smpp/receiver_test.go`, `smpp/transceiver_test.go`
- `smpp/server_test.go`, `smpp/smpptest/server_test.go`
- `smpp/pdu/header_test.go`, `smpp/pdu/types_test.go`, `smpp/pdu/factory_test.go`
- `smpp/pdu/pdufield/*_test.go`, `smpp/pdu/pdutext/*_test.go`, `smpp/pdu/pdutlv/*_test.go`
- `smpp/encoding/gsm7_test.go`

## Padroes de Tratamento de Erros

### Sentinela de erros
Erros globais pre-definidos para condicoes conhecidas:

- `ErrNotConnected` — tentativa de uso em conexao morta
- `ErrNotBound` — operacao antes de Bind()
- `ErrTimeout` — timeout aguardando resposta
- `ErrMaxWindowSize` — janela de envio esgotada

### Status SMPP como Error
`pdu.Status` implementa `error` interface (`header.go:153`), permitindo retornar codigos de erro SMPP diretamente como erros Go:

```go
if s := resp.PDU.Header().Status; s != 0 {
    return sm, s  // s implementa error
}
```

### Concorrencia e sincronizacao
- `sync.Mutex` para proteger estado compartilhado (maps de inflight, conn switch)
- `sync.RWMutex` para leitura de timestamps de EnquireLink
- `sync/atomic` para contagem de inflight messages
- `sync.Once` para garantir Close() idempotente
