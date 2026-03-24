# Estrutura do Codebase

**Data de Análise:** 2026-03-24

## Layout de Diretórios

```
go-smpp/
├── cmd/
│   ├── sms/              # CLI smppcli — envio e consulta de SMS
│   │   └── main.go
│   └── smsapid/          # Placeholder — não implementado
├── smpp/                 # Pacote principal da biblioteca
│   ├── client.go         # Conexão persistente com reconnect e backoff
│   ├── conn.go           # Abstrações de conexão TCP/TLS (Conn, connSwitch)
│   ├── doc.go            # Documentação do pacote
│   ├── receiver.go       # Cliente SMPP Receiver (recepção de SMS)
│   ├── server.go         # Servidor SMPP com sessions e handlers
│   ├── transceiver.go    # Cliente SMPP Transceiver (envio + recepção)
│   ├── transmitter.go    # Cliente SMPP Transmitter (envio de SMS)
│   ├── encoding/         # Codificação GSM 7-bit
│   │   └── gsm7.go
│   ├── pdu/              # Protocol Data Units — serialização binária
│   │   ├── body.go       # Interface Body (abstração de PDU)
│   │   ├── codec.go      # Codec base + Decode/SerializeTo
│   │   ├── factory.go    # Factory com sequence number isolado
│   │   ├── header.go     # Header 16 bytes + Status codes
│   │   ├── types.go      # Definições de todos os tipos de PDU
│   │   ├── pdufield/     # Campos obrigatórios de PDU
│   │   │   ├── body.go   # Interface e factory de campos
│   │   │   ├── list.go   # Lista ordenada para decode
│   │   │   ├── map.go    # Mapa de campos com Set/Get
│   │   │   └── types.go  # Tipos: Fixed, Variable, SM, UDH
│   │   ├── pdutext/      # Codecs de texto
│   │   │   ├── codec.go  # Interface Codec + routing por DataCoding
│   │   │   ├── gsm7.go   # GSM 7-bit (SMSC Default)
│   │   │   ├── latin1.go # Latin1 / Windows-1252
│   │   │   ├── ucs2.go   # UCS-2 / UTF-16-BE
│   │   │   ├── iso88591.go # ISO-8859-1
│   │   │   ├── iso88595.go # ISO-8859-5 (Cyrillic)
│   │   │   └── raw.go    # Sem codificação (passthrough)
│   │   └── pdutlv/       # Campos opcionais TLV
│   │       ├── tlv_types.go    # Tags padrão SMPP (50+)
│   │       ├── tlv_body.go     # Decode de TLV individual
│   │       ├── tlv_list.go     # Lista de TLVs
│   │       ├── tlv_map.go      # Mapa de TLVs
│   │       └── messagestate.go # Estados de mensagem
│   └── smpptest/         # Servidor SMPP de teste
│       ├── conn.go       # Conexão server-side
│       ├── doc.go        # Documentação do pacote
│       └── server.go     # Server mock com echo handler
├── docs/                 # Documentação técnica
├── specs/                # Especificações técnicas
├── go.mod                # Definição do módulo Go
└── go.sum                # Checksums de dependências
```

## Onde Colocar Código Novo

**Novo tipo de PDU (ex: AlertNotification):**
1. Defina o ID como constante em `smpp/pdu/types.go` (já existe para os pendentes)
2. Crie o constructor `newAlertNotification(hdr)` e `NewAlertNotification()` em `smpp/pdu/types.go`
3. Adicione o case no switch de `Decode()` em `smpp/pdu/codec.go:178`
4. Adicione o case no switch de `CreatePDU()` em `smpp/pdu/factory.go`
5. Crie testes em `smpp/pdu/factory_test.go` e `smpp/pdu/types_test.go`

**Novo campo de PDU (pdufield):**
1. Defina a constante `Name` em `smpp/pdu/pdufield/types.go`
2. Adicione o case de decode em `smpp/pdu/pdufield/body.go`
3. Se for um tipo novo, crie o struct que implementa `Body` interface em `smpp/pdu/pdufield/types.go`
4. Crie testes em `smpp/pdu/pdufield/types_test.go`

**Novo codec de texto (ex: UTF-8):**
1. Crie `smpp/pdu/pdutext/utf8.go` com tipo que implementa `pdutext.Codec`
2. Defina a constante `DataCoding` em `smpp/pdu/pdutext/codec.go`
3. Adicione cases em `pdutext.Encode()` e `pdutext.Decode()` em `smpp/pdu/pdutext/codec.go`
4. Crie testes em `smpp/pdu/pdutext/utf8_test.go`

**Nova tag TLV:**
1. Defina a constante `Tag` em `smpp/pdu/pdutlv/tlv_types.go`
2. Crie testes em `smpp/pdu/pdutlv/tlv_types_test.go`

**Novo comando CLI:**
1. Defina o `cli.Command` em `cmd/sms/main.go`
2. Adicione à lista `app.Commands` em `cmd/sms/main.go:64`

**Nova funcionalidade de client (ex: novo método no Transmitter):**
1. Adicione o método em `smpp/transmitter.go`
2. Se precisa de PDU novo, implemente-o primeiro (ver acima)
3. Crie testes em `smpp/transmitter_test.go` usando `smpptest.Server`

**Novo teste:**
- Coloque no arquivo `*_test.go` do mesmo pacote e diretório do código testado
- Use `smpptest.NewUnstartedServer()` para testes que precisam de um servidor SMPP

## Naming de Arquivos

**Arquivos:** Use snake_case (`gsm7.go`, `tlv_types.go`, `tlv_body.go`)
**Diretórios:** Use lowercase sem separador (`pdufield/`, `pdutext/`, `pdutlv/`, `smpptest/`)
**Testes:** Use `[nome]_test.go` ao lado do módulo (`transmitter_test.go`, `gsm7_test.go`)
**Docs do pacote:** Use `doc.go` para documentação do pacote Go

## Diretórios Especiais

**`smpp/smpptest/`:**
- Propósito: servidor SMPP de teste — NÃO é para produção
- Gerado automaticamente: Não
- No git: Sim

**`cmd/smsapid/`:**
- Propósito: placeholder para um servidor de API SMS (não implementado)
- Status: vazio, sem código

---

*Análise de estrutura: 2026-03-24*
