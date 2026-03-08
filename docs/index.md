# Documentacao do go-smpp

## Visao Geral

O `go-smpp` e uma biblioteca Go que implementa o protocolo SMPP 3.4 (Short Message Peer-to-Peer), utilizado para envio e recebimento de mensagens SMS entre aplicacoes (ESMEs) e centros de servico de mensagens curtas (SMSCs).

A biblioteca fornece abstraccoes de alto nivel para os tres modos de operacao SMPP — Transmitter (envio), Receiver (recebimento) e Transceiver (ambos) — com gerenciamento automatico de conexao, reconexao com backoff exponencial, rate limiting e suporte a mensagens longas (concatenadas via UDH). Inclui tambem um servidor SMPP completo para testes e um servidor de producao com sessoes e handlers customizaveis.

Modulo Go: `github.com/devyx-tech/go-smpp`

## Documentacao Disponivel

### Arquitetura e Stack
- [Stack Tecnologica](stack.md) — Tecnologias, frameworks e ferramentas utilizadas
- [Padroes de Design](patterns.md) — Padroes arquiteturais e de codigo

### Funcionalidades e Regras
- [Funcionalidades](features.md) — Descricao das funcionalidades principais
- [Regras de Negocio](business-rules.md) — Regras do protocolo SMPP implementadas

### Integracoes
- [Integracoes](integrations.md) — Comunicacao com SMSCs e dependencias externas

## Links Rapidos
- Repositorio: `github.com/devyx-tech/go-smpp`
- Licenca: MIT
- Versao do protocolo: SMPP 3.4
- Go minimo: 1.18
