# Documentação do go-smpp

## Visão Geral

**go-smpp** é uma biblioteca Go open-source que implementa o protocolo **SMPP 3.4** (Short Message Peer-to-Peer) para envio e recebimento de SMS. Esta biblioteca fornece uma implementação completa, robusta e pronta para produção do protocolo SMPP, com suporte a múltiplos encodings de caracteres, reconexão automática, tratamento de mensagens longas e muito mais.

### Propósito
Fornecer uma interface Go limpa e idiomática para comunicação com SMSCs (Short Message Service Centers) através do protocolo SMPP, permitindo que aplicações Go enviem e recebam mensagens SMS de forma confiável.

### Características Principais
- Implementação completa do protocolo SMPP 3.4
- Suporte a Transmitter, Receiver e Transceiver
- Reconexão automática com exponential backoff
- Suporte a múltiplos encodings (GSM7, Latin1, UCS2, ISO-8859-5, Raw)
- Tratamento automático de mensagens longas (splitting e merging)
- EnquireLink keepalive para manutenção de conexões
- Interface baseada em canais (channels) para comunicação assíncrona
- Rate limiting opcional
- Suporte a TLS
- CLI tools para teste e uso standalone

### Estatísticas do Projeto
- **Linguagem**: Go 1.18+
- **Linhas de Código**: ~7.815
- **Arquivos Go**: 62
- **Pacotes**: 8 principais
- **Licença**: BSD-style
- **Uso**: Biblioteca importada em outros serviços Go

---

## Documentação Disponível

### Arquitetura e Stack
- [**Stack Tecnológica**](stack.md) - Tecnologias, frameworks, dependências e arquitetura geral
- [**Padrões de Design**](patterns.md) - Padrões arquiteturais, organização de código e convenções

### Funcionalidades e Regras
- [**Funcionalidades**](features.md) - Descrição detalhada das funcionalidades principais (Transmitter, Receiver, Encodings, CLI)
- [**Regras de Negócio**](business-rules.md) - Regras do protocolo SMPP implementadas, validações e tratamento de erros

### Integrações
- [**Guia de Integração**](integrations.md) - Como integrar a biblioteca em projetos Go, exemplos de uso e dependências

---

## Início Rápido

### Instalação
```bash
go get github.com/devyx-tech/go-smpp
```

### Exemplo Básico - Enviar SMS
```go
package main

import (
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdutext"
)

func main() {
    tx := &smpp.Transmitter{
        Addr:   "localhost:2775",
        User:   "username",
        Passwd: "password",
    }

    conn := tx.Bind()
    status := <-conn // aguarda conexão

    if status.Error() != nil {
        panic(status.Error())
    }

    sm, err := tx.Submit(&smpp.ShortMessage{
        Src:  "1234",
        Dst:  "447911111111",
        Text: pdutext.Raw("Hello World"),
    })

    if err != nil {
        panic(err)
    }

    defer tx.Close()
}
```

### Exemplo Básico - Receber SMS
```go
package main

import (
    "fmt"
    "github.com/devyx-tech/go-smpp/smpp"
    "github.com/devyx-tech/go-smpp/smpp/pdu"
)

func main() {
    rx := &smpp.Receiver{
        Addr:   "localhost:2775",
        User:   "username",
        Passwd: "password",
        Handler: func(p pdu.Body) {
            switch p.Header().ID {
            case pdu.DeliverSMID:
                fields := p.Fields()
                fmt.Printf("SMS recebido: %s\n", fields[pdufield.ShortMessage])
            }
        },
    }

    conn := rx.Bind()
    <-conn // aguarda conexão

    // Mantém o receiver rodando
    select {}
}
```

---

## Estrutura do Repositório

```
go-smpp/
├── cmd/                          # Ferramentas CLI
│   ├── sms/                      # Cliente SMS CLI
│   └── smsapid/                  # Daemon API SMS
├── smpp/                         # Pacote principal SMPP
│   ├── encoding/                 # Encodings de caracteres (GSM7)
│   ├── pdu/                      # Protocol Data Units
│   │   ├── pdufield/             # Campos de PDU
│   │   ├── pdutext/              # Codecs de texto
│   │   └── pdutlv/               # Tag-Length-Value fields
│   ├── smpptest/                 # Servidor de teste SMPP
│   ├── client.go                 # Gestão de conexão cliente
│   ├── conn.go                   # Conexão de baixo nível
│   ├── transmitter.go            # Implementação Transmitter
│   ├── receiver.go               # Implementação Receiver
│   ├── transceiver.go            # Implementação Transceiver
│   └── server.go                 # Servidor SMPP para testes
├── docs/                         # Documentação (esta pasta)
├── go.mod                        # Dependências Go
└── README.md                     # Documentação principal
```

---

## Principais Conceitos SMPP

### Tipos de Conexão
- **Transmitter**: Conexão para **envio** de SMS (client → server)
- **Receiver**: Conexão para **recebimento** de SMS (server → client)
- **Transceiver**: Conexão **bidirecional** (envio e recebimento simultâneo)

### PDU (Protocol Data Unit)
Unidade básica de comunicação SMPP. Cada operação (bind, submit, deliver, etc.) é representada por um tipo de PDU específico.

### Encodings
- **GSM7**: Encoding padrão GSM 03.38 (160 caracteres por SMS)
- **Latin1**: ISO-8859-1 para caracteres ocidentais (70 caracteres)
- **UCS2**: Unicode para caracteres internacionais (70 caracteres)
- **ISO-8859-5**: Cirílico (70 caracteres)
- **Raw**: Bytes brutos sem transformação

### Mensagens Longas
Mensagens com mais de 160 caracteres (GSM7) ou 70 caracteres (UCS2/Latin1) são automaticamente divididas em múltiplas partes usando UDH (User Data Header) headers.

---

## Links Úteis

- **Repositório**: https://github.com/devyx-tech/go-smpp
- **Especificação SMPP 3.4**: [SMPP Protocol Specification](http://www.smsforum.net/)
- **Go Package Doc**: Consulte o código-fonte para documentação inline

---

## Contribuindo

Este é um projeto open-source de uso interno. Para contribuir:
1. Fork o repositório
2. Crie uma branch para sua feature (`git checkout -b feature/nova-funcionalidade`)
3. Commit suas mudanças (`git commit -m 'Adiciona nova funcionalidade'`)
4. Push para a branch (`git push origin feature/nova-funcionalidade`)
5. Abra um Pull Request

---

## Licença

Este projeto está licenciado sob uma licença BSD-style. Veja o arquivo `LICENSE` para mais detalhes.
