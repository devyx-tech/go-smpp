# Regras de Negocio

## Contexto

As "regras de negocio" deste repositorio sao definidas pela especificacao SMPP 3.4 (Short Message Peer-to-Peer Protocol). O protocolo define como aplicacoes (ESMEs) se comunicam com centros de servico de mensagens (SMSCs) para envio e recebimento de SMS.

## Regras Criticas

### 1. Binding obrigatorio antes de qualquer operacao
- **Descricao**: Toda comunicacao SMPP exige que o cliente primeiro estabeleca um bind (BindTransmitter, BindReceiver ou BindTransceiver) antes de enviar/receber mensagens.
- **Justificativa**: Especificacao SMPP 3.4 — o bind autentica e define o modo de operacao.
- **Implementacao**: `smpp/transmitter.go:70` (Bind), `smpp/conn.go:25` (`ErrNotBound`)
- **Validacoes**: Operacoes como `Submit()` verificam se o bind foi realizado; retornam `ErrNotBound` caso contrario.

### 2. Interface Version fixa em 0x34
- **Descricao**: O campo `interface_version` e sempre definido como `0x34` (52 decimal) durante o bind.
- **Justificativa**: Indica ao SMSC que o cliente suporta SMPP versao 3.4.
- **Implementacao**: `smpp/client.go:296` — `bind()` seta `pdufield.InterfaceVersion` para `0x34`.

### 3. Sequence number unico por PDU
- **Descricao**: Cada PDU enviado recebe um sequence number unico que e usado para correlacionar requisicoes com respostas.
- **Justificativa**: O SMPP e um protocolo assincronico — multiplas requisicoes podem estar em voo simultaneamente.
- **Implementacao**: Dois mecanismos:
  - Contador atomico global: `smpp/pdu/codec.go:19` (`nextSeq`)
  - Factory isolada: `smpp/pdu/factory.go:62-69` com wrap em `0x7FFFFFFF`
- **Validacoes**: O Transmitter mantem um mapa de `inflight` (`transmitter.go:58`) para correlacionar respostas pelo seq number.

### 4. Limite maximo de PDU: 4096 bytes
- **Descricao**: Nenhum PDU pode exceder 4096 bytes.
- **Justificativa**: Definido na especificacao SMPP 3.4 como tamanho maximo de PDU.
- **Implementacao**: `smpp/pdu/body.go:15` — `MaxSize = 4096`
- **Validacoes**: `DecodeHeader()` em `header.go:130` rejeita PDUs maiores que `MaxSize`.

### 5. Header de PDU: exatamente 16 bytes
- **Descricao**: Todo PDU tem um header de 16 bytes contendo Length (4), Command ID (4), Status (4) e Sequence Number (4), em big-endian.
- **Justificativa**: Formato definido pela especificacao SMPP 3.4.
- **Implementacao**: `smpp/pdu/header.go:108` — `HeaderLen = 16`
- **Validacoes**: PDUs menores que 16 bytes sao rejeitados (`header.go:127`).

### 6. Mensagem curta limitada a 140 bytes
- **Descricao**: O campo `short_message` tem um limite de 140 octetos. Mensagens maiores devem ser fragmentadas via UDH.
- **Justificativa**: Limite definido pelo protocolo SMPP/GSM.
- **Implementacao**: `smpp/transmitter.go:334` — `maxLen = 134` (140 - 6 bytes de UDH header)
- **Excepcoes**: Pode-se usar `message_payload` TLV para payloads maiores (nao implementado neste repo).

### 7. Maximo de 254 destinatarios em SubmitMulti
- **Descricao**: O PDU SubmitMulti suporta no maximo 254 enderecos de destino.
- **Justificativa**: Limite definido pela especificacao SMPP 3.4 (campo `number_of_dests` e uint8).
- **Implementacao**: `smpp/transmitter.go:30` — `MaxDestinationAddress = 254`
- **Validacoes**: `submitMsgMulti()` em `transmitter.go:450-453` verifica o limite e retorna erro.

## Validacoes e Restricoes

### Autenticacao
- O servidor valida `system_id` e `password` no bind PDU
- Credenciais invalidas resultam em `InvalidSystemID` (0x0000000f) ou `InvalidPassword` (0x0000000e)
- **Implementacao**: `smpp/server.go:258-293`

### Status de resposta SMPP
Todas as respostas PDU carregam um status code. A biblioteca mapeia 30+ codigos de status SMPP para mensagens de erro Go:
- `OK` (0x00000000) — sucesso
- `InvalidMessageLength` (0x00000001)
- `BindFailed` (0x0000000d)
- `MessageQueueFull` (0x00000014)
- `ThrottlingError` (0x00000058)
- Lista completa em `smpp/pdu/header.go:22-69`

### Codificacao de texto (DataCoding)
- O campo `data_coding` e configurado automaticamente quando `ShortMessage` recebe um `pdutext.Codec`
- O campo `sm_length` e calculado automaticamente apos o encode do texto
- **Implementacao**: `smpp/pdu/pdufield/map.go:46-57`

## Politicas e Workflows

### Workflow de Conexao
1. Client chama `Bind()` — retorna channel de status
2. `client.Bind()` inicia loop de conexao em goroutine
3. Dial TCP/TLS para o SMSC
4. Se falha: notifica `ConnectionFailed`, retry com backoff exponencial
5. Envia bind PDU (BindTransmitter/Receiver/Transceiver)
6. Se falha: notifica `BindFailed`, retry
7. Se sucesso: notifica `Connected`, inicia enquireLink e handlePDU
8. Se conexao cai: notifica `Disconnected`, retry

### Workflow de Envio (Submit)
1. Verifica se bind foi realizado (`ErrNotBound`)
2. Verifica window size (`ErrMaxWindowSize`)
3. Registra request no mapa inflight com seq number
4. Aplica rate limiter (se configurado)
5. Serializa e envia PDU
6. Aguarda resposta no channel (com timeout)
7. Valida response PDU ID e status
8. Retorna `ShortMessage` com resposta

### Workflow de EnquireLink
1. Envia EnquireLink periodicamente (intervalo configuravel, minimo 10s)
2. Registra timestamp de ultima resposta recebida
3. Se tempo desde ultima resposta > `EnquireLinkTimeout` (default 3x intervalo):
   - Envia Unbind
   - Fecha conexao
   - Trigger reconnect

### Workflow de Mensagens Longas (Recebimento)
1. Recebe DeliverSM com UDH flag no ESMClass
2. Decodifica UDH header (IEI 0x00 = concatenated short messages)
3. Armazena parte em `MergeHolder` indexado por message ID
4. Quando todas as partes chegam: ordena por part ID, concatena, chama handler
5. Cleanup periodico remove partes expiradas (`MergeInterval`)

## Regras de Dominio

### Modos de Delivery Receipt
- `NoDeliveryReceipt` (0x00) — Sem confirmacao de entrega
- `FinalDeliveryReceipt` (0x01) — Confirmacao final de entrega
- `FailureDeliveryReceipt` (0x02) — Apenas em caso de falha

### Estados de Mensagem (QuerySM)
| Codigo | Estado | Descricao |
|---|---|---|
| 0 | SCHEDULED | Agendada para envio |
| 1 | ENROUTE | Em transito |
| 2 | DELIVERED | Entregue |
| 3 | EXPIRED | Expirada |
| 4 | DELETED | Deletada |
| 5 | UNDELIVERABLE | Nao entregavel |
| 6 | ACCEPTED | Aceita |
| 7 | UNKNOWN | Estado desconhecido |
| 8 | REJECTED | Rejeitada |
| 9 | SKIPPED | Ignorada |

### Validade de Mensagem
- Formato SMPP absoluto: `YYMMDDhhmmsstnnp` (spec 7.1.1)
- Calculada como `now().UTC().Add(validity)`, formatada com sufixo `"000+"`
- **Implementacao**: `smpp/transmitter.go:582-586`

### Tipos de Destino em SubmitMulti
- `0x01` — SME Address (endereco de telefone)
- `0x02` — Distribution List (lista de distribuicao)
- **Implementacao**: `smpp/transmitter.go:458-474`
