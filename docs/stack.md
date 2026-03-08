# Stack Tecnologica

## Linguagens e Runtime

| Tecnologia | Versao | Papel |
|---|---|---|
| Go | >= 1.18 | Linguagem principal |

## Frameworks Principais

Nenhum framework web ou aplicacional e utilizado. O projeto e uma biblioteca Go pura construida sobre a standard library, com dependencias minimas.

## Bibliotecas Chave

| Biblioteca | Versao | Proposito |
|---|---|---|
| `golang.org/x/text` | v0.10.0 | Codificacao de texto (UCS2/UTF-16, Latin1/Windows-1252, ISO-8859-5) |
| `golang.org/x/net` | v0.11.0 | `context.Context` para rate limiting |
| `golang.org/x/time` | v0.3.0 | `rate.Limiter` para controle de taxa de envio |
| `github.com/urfave/cli` | v1.22.14 | CLI para a ferramenta `smppcli` (somente `cmd/sms`) |

## Banco de Dados

Nao aplicavel. A biblioteca nao utiliza banco de dados — opera exclusivamente sobre conexoes TCP com SMSCs.

## Infraestrutura

- **Protocolo de rede**: TCP (com suporte opcional a TLS)
- **Porta padrao**: 2775 (convencao SMPP)
- **CI**: Travis CI (`.travis.yml` configurado para Go 1.9.4, legado)
- **GitHub Actions**: Workflows para code review e PR assistant (`.github/workflows/`)

## Ferramentas de Desenvolvimento

| Ferramenta | Proposito |
|---|---|
| `go test` | Testes unitarios (padrao Go) |
| Travis CI | Integracao continua (legado) |
| GitHub Actions | CI/CD moderno |

## Arquitetura Geral

A biblioteca segue uma arquitetura em camadas:

```
cmd/                  CLI tools (consumidores da biblioteca)
  sms/                smppcli - cliente de linha de comando
  smsapid/            [nao implementado]

smpp/                 Camada de aplicacao (alto nivel)
  client.go           Gerenciamento de conexao persistente
  conn.go             Abstraccao de conexao TCP/TLS
  transmitter.go      Modo Transmitter (envio de SMS)
  receiver.go         Modo Receiver (recebimento de SMS)
  transceiver.go      Modo Transceiver (envio + recebimento)
  server.go           Servidor SMPP de producao

  pdu/                Camada de protocolo (codecs PDU)
    codec.go          Serializacao/deserializacao de PDUs
    header.go         Header de 16 bytes (Len, ID, Status, Seq)
    types.go          Definicoes de todos os tipos de PDU
    factory.go        Factory com controle de sequence numbers

    pdufield/         Campos obrigatorios de PDUs
      types.go        Tipos de campo (Fixed, Variable, SM, UDH)
      list.go         Decodificacao ordenada de campos
      map.go          Mapa de campos com Set/Get tipado
      body.go         Interface e factory de campos

    pdutext/          Codecs de texto para mensagens
      codec.go        Interface Codec e routing por DataCoding
      gsm7.go         GSM 7-bit (SMSC Default Alphabet)
      latin1.go       Latin1 / Windows-1252
      ucs2.go         UCS-2 / UTF-16-BE
      iso88591.go     ISO-8859-1
      iso88595.go     ISO-8859-5 (Cyrillic)
      raw.go          Sem codificacao

    pdutlv/           Campos opcionais TLV (Tag-Length-Value)
      tlv_types.go    Tags padrao SMPP (50+ tags)
      tlv_body.go     Decodificacao de TLV
      tlv_map.go      Mapa de campos TLV
      messagestate.go Estados de mensagem

  encoding/           Codificacao GSM 7-bit
    gsm7.go           Encoder/Decoder GSM 7-bit (packed/unpacked)

  smpptest/           Servidor de teste
    server.go         Servidor mock com echo handler
    conn.go           Conexao server-side para testes
```

## Decisoes Arquiteturais Importantes

### Conexao persistente com reconnect automatico
O `client` implementa um loop de reconnect com backoff exponencial (fator `e`, maximo 120s). Isso garante resiliencia sem intervencao manual. A opcao `BindInterval` permite fixar um intervalo constante.

### Separacao Transmitter / Receiver / Transceiver
Segue a especificacao SMPP 3.4 que define tres modos de binding. O `Transceiver` compoe o `Transmitter` via embedding, evitando duplicacao de logica de envio.

### PDU como interface (Body)
Todas as PDUs implementam `pdu.Body`, permitindo serializacao, deserializacao e manipulacao generica. O `Codec` base implementa a logica comum e cada tipo de PDU define apenas seus campos.

### Sequence number global vs Factory
Ha dois mecanismos: um contador atomico global (`nextSeq` em `codec.go`) e uma `Factory` com contador isolado por instancia. A Factory e recomendada para cenarios com multiplas conexoes para evitar colisoes.

### Codificacao de texto via interface Codec
`pdutext.Codec` abstrai codificacao/decodificacao de texto. O campo `DataCoding` do PDU e definido automaticamente ao usar `Map.Set(ShortMessage, codec)`.
