# Testes

**Data de Análise:** 2026-03-24

## Infraestrutura

**Framework:** `testing` (stdlib Go)
**Config:** Nenhum arquivo de configuração de testes — usa padrões Go
**Rodar testes:** `go test ./...`
**Rodar com coverage:** `go test -cover ./...`
**Rodar teste específico:** `go test -run TestShortMessage ./smpp/`

## Onde Colocar Testes

**Localização:** Arquivo `*_test.go` no mesmo pacote e diretório do código testado
**Naming:** `[nome]_test.go` — ex: `transmitter_test.go`, `gsm7_test.go`
**Exemplos:** `example_test.go` no pacote `smpp` para exemplos documentados

**Estrutura:**
```
smpp/
├── conn_test.go            # Testes de conexão
├── transmitter_test.go     # Testes do Transmitter
├── receiver_test.go        # Testes do Receiver
├── transceiver_test.go     # Testes do Transceiver
├── server_test.go          # Testes do Server
├── example_test.go         # Exemplos documentados (godoc)
├── encoding/
│   └── gsm7_test.go        # Testes GSM7 encoding
├── pdu/
│   ├── header_test.go
│   ├── types_test.go
│   ├── factory_test.go
│   ├── pdufield/
│   │   ├── body_test.go
│   │   ├── list_test.go
│   │   ├── map_test.go
│   │   └── types_test.go
│   ├── pdutext/
│   │   ├── codec_test.go
│   │   ├── gsm7_test.go
│   │   ├── latin1_test.go
│   │   ├── ucs2_test.go
│   │   ├── iso88591_test.go
│   │   ├── iso88595_test.go
│   │   └── raw_test.go
│   └── pdutlv/
│       ├── tlv_body_test.go
│       ├── tlv_list_test.go
│       ├── tlv_map_test.go
│       └── tlv_types_test.go
└── smpptest/
    └── server_test.go      # Testes do servidor de teste
```

## Padrão de Teste

Use este padrão ao criar novos testes. O padrão base: criar `smpptest.Server` com handler customizado, conectar `Transmitter`/`Receiver`, exercitar operação, verificar resultado.

```go
// Extraído de smpp/transmitter_test.go:19-63
func TestShortMessage(t *testing.T) {
    s := smpptest.NewUnstartedServer()
    s.Handler = func(c smpptest.Conn, p pdu.Body) {
        switch p.Header().ID {
        case pdu.SubmitSMID:
            r := pdu.NewSubmitSMResp()
            r.Header().Seq = p.Header().Seq
            r.Fields().Set(pdufield.MessageID, "foobar")
            c.Write(r)
        default:
            smpptest.EchoHandler(c, p)
        }
    }
    s.Start()
    defer s.Close()
    tx := &Transmitter{
        Addr:        s.Addr(),
        User:        smpptest.DefaultUser,
        Passwd:      smpptest.DefaultPasswd,
        RateLimiter: rate.NewLimiter(rate.Limit(10), 1),
    }
    defer tx.Close()
    conn := <-tx.Bind()
    switch conn.Status() {
    case Connected:
    default:
        t.Fatal(conn.Error())
    }
    sm, err := tx.Submit(&ShortMessage{
        Src:      "root",
        Dst:      "foobar",
        Text:     pdutext.Raw("Lorem ipsum"),
        Validity: 10 * time.Minute,
        Register: pdufield.NoDeliveryReceipt,
    })
    if err != nil {
        t.Fatal(err)
    }
    msgid := sm.RespID()
    if msgid == "" {
        t.Fatalf("pdu does not contain msgid: %#v", sm.Resp())
    }
    if msgid != "foobar" {
        t.Fatalf("unexpected msgid: want foobar, have %q", msgid)
    }
}
```

## Mocking

**Abordagem:** Servidor SMPP real (`smpptest.Server`) em vez de mocks de interface. Cada teste inicia um servidor local, conecta o cliente, e verifica o comportamento end-to-end.

**Biblioteca:** `smpp/smpptest` — servidor SMPP leve que roda em `127.0.0.1` com porta aleatória.

**Setup padrão:**
```go
// Criar servidor com handler customizado
s := smpptest.NewUnstartedServer()
s.Handler = func(c smpptest.Conn, p pdu.Body) {
    // lógica de resposta customizada
}
s.Start()
defer s.Close()

// Conectar cliente ao servidor
tx := &Transmitter{
    Addr:   s.Addr(),
    User:   smpptest.DefaultUser,  // "client"
    Passwd: smpptest.DefaultPasswd, // "secret"
}
conn := <-tx.Bind()
```

## Fixtures e Test Data

**Localização:** Não há diretório de fixtures separado. Dados de teste são definidos inline nos testes.
**Credenciais padrão de teste:** `smpptest.DefaultUser = "client"`, `smpptest.DefaultPasswd = "secret"`
**System ID padrão:** `smpptest.DefaultSystemID = "smpptest"`

## Cobertura

**Ferramenta:** `go test -cover` (builtin)
**Threshold:** Não definido

## Tipos de Teste Presentes

- **Unitários:** Presentes — `smpp/pdu/pdufield/*_test.go`, `smpp/pdu/pdutext/*_test.go`, `smpp/pdu/pdutlv/*_test.go`, `smpp/encoding/gsm7_test.go`
- **Integração:** Presentes — `smpp/transmitter_test.go`, `smpp/receiver_test.go`, `smpp/transceiver_test.go`, `smpp/server_test.go` (usam `smpptest.Server`)
- **Exemplos:** Presentes — `smpp/example_test.go` (exemplos para godoc)
- **E2E:** Ausentes — não há testes contra SMSCs reais

---

*Análise de testes: 2026-03-24*
