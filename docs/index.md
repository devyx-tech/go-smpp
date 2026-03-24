# go-smpp

**Data de Análise:** 2026-03-24

## Visão Geral

go-smpp é uma biblioteca Go que implementa o protocolo SMPP 3.4 (Short Message Peer-to-Peer). Fornece client e server SMPP com suporte a conexões persistentes, reconexão automática com backoff exponencial, rate limiting, TLS, e múltiplos encodings de texto (GSM7, UCS2, Latin1, ISO-8859-5).

A biblioteca é consumida como dependência Go por aplicações que precisam enviar ou receber SMS via SMSC (Short Message Service Center). Inclui também uma ferramenta CLI (`cmd/sms/`) para envio e consulta de mensagens via linha de comando.

## Documentação

### Arquitetura e Stack
- [Stack Tecnológica](stack.md) — Go 1.18, dependências, runtime
- [Arquitetura](architecture.md) — Camadas PDU, client, server, fluxo de dados
- [Estrutura](structure.md) — Layout de diretórios, onde colocar código novo

### Convenções e Testes
- [Convenções](conventions.md) — Padrões de código prescritivos
- [Testes](testing.md) — Infraestrutura e padrões de teste com smpptest

### Funcionalidades e Regras
- [Funcionalidades](features.md) — Transmitter, Receiver, Transceiver, Server, encodings
- [Regras de Negócio](business-rules.md) — Regras do protocolo SMPP 3.4, validações, encoding

### Integrações e Saúde
- [Integrações](integrations.md) — Conexões TCP/TLS, rate limiting
- [Preocupações](concerns.md) — PDUs não implementados, tech debt

## Links Rápidos
- Repositório: `github.com/devyx-tech/go-smpp`
- Import: `github.com/devyx-tech/go-smpp/smpp`
- CLI: `go run ./cmd/sms/`
- Testes: `go test ./...`
