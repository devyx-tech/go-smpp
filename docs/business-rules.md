# Regras de Negócio

**Data de Análise:** 2026-03-24

## Regras Críticas

### Binding obrigatório antes de qualquer operação

**Regra:** Toda comunicação SMPP exige um bind (BindTransmitter, BindReceiver ou BindTransceiver) antes de enviar ou receber mensagens.
**Justificativa:** Especificação SMPP 3.4 — o bind autentica o cliente e define o modo de operação.
**Implementação:** `smpp/transmitter.go:70` (Bind), `smpp/conn.go:25` (`ErrNotBound`)
**Validações:** `Submit()` e `QuerySM()` verificam se o bind foi realizado; retornam `ErrNotBound` caso contrário.

### Interface Version fixa em 0x34

**Regra:** Defina `interface_version` como `0x34` (52 decimal) em todo bind PDU.
**Justificativa:** Indica ao SMSC que o cliente suporta SMPP versão 3.4.
**Implementação:** `smpp/client.go:296` — `bind()` seta `pdufield.InterfaceVersion` para `0x34`.

### Sequence number único por PDU

**Regra:** Cada PDU enviado recebe um sequence number único para correlacionar requisições com respostas.
**Justificativa:** SMPP é protocolo assíncrono — múltiplas requisições em voo simultaneamente.
**Implementação:**
- Contador atômico global: `smpp/pdu/codec.go:19` (`nextSeq`)
- Factory isolada: `smpp/pdu/factory.go:62-69` com wrap em `0x7FFFFFFF`
**Validações:** Transmitter mantém mapa `inflight` (`smpp/transmitter.go:58`) para correlação.

### Limite máximo de PDU: 4096 bytes

**Regra:** Nenhum PDU pode exceder 4096 bytes.
**Justificativa:** Definido na especificação SMPP 3.4.
**Implementação:** `smpp/pdu/body.go:15` — `MaxSize = 4096`
**Validações:** `DecodeHeader()` em `smpp/pdu/header.go:130` rejeita PDUs maiores que `MaxSize`.

### Header de PDU: exatamente 16 bytes

**Regra:** Todo PDU tem header de 16 bytes: Length (4) + Command ID (4) + Status (4) + Sequence Number (4), big-endian.
**Justificativa:** Formato definido pela especificação SMPP 3.4.
**Implementação:** `smpp/pdu/header.go:108` — `HeaderLen = 16`
**Validações:** PDUs menores que 16 bytes são rejeitados em `smpp/pdu/header.go:127`.

### Mensagem curta limitada a 140 bytes

**Regra:** O campo `short_message` tem limite de 140 octetos. Mensagens maiores devem ser fragmentadas via UDH.
**Justificativa:** Limite do protocolo SMPP/GSM.
**Implementação:** `smpp/transmitter.go:334` — `maxLen = 134` (140 - 6 bytes de UDH header)
**Exceções:** O TLV `message_payload` permite payloads maiores (não implementado neste repositório).

### Máximo de 254 destinatários em SubmitMulti

**Regra:** SubmitMulti suporta no máximo 254 endereços de destino.
**Justificativa:** Campo `number_of_dests` é uint8 (max 254 na spec SMPP 3.4).
**Implementação:** `smpp/transmitter.go:30` — `MaxDestinationAddress = 254`
**Validações:** `submitMsgMulti()` em `smpp/transmitter.go:450-453` verifica e retorna erro.

## Validações e Restrições

- **Autenticação do servidor:** Valida `system_id` e `password` no bind PDU. Credenciais inválidas resultam em `InvalidSystemID` (0x0F) ou `InvalidPassword` (0x0E) — `smpp/server.go:258-293`
- **DataCoding automático:** Configurado automaticamente quando `ShortMessage` recebe um `pdutext.Codec` — `smpp/pdu/pdufield/map.go:46-57`
- **sm_length automático:** Calculado automaticamente após encode do texto — `smpp/pdu/pdufield/map.go`
- **EnquireLink mínimo 10s:** O intervalo de EnquireLink não pode ser menor que 10 segundos — `smpp/client.go:121-123`
- **Window size:** Se configurado, rejeita envio quando mensagens inflight excedem o limite — `smpp/transmitter.go:283-289`

## Políticas e Workflows

### Workflow de Conexão

1. Client chama `Bind()` — retorna channel de status
2. `client.Bind()` inicia loop de conexão em goroutine
3. Dial TCP/TLS para o SMSC
4. Se falha: notifica `ConnectionFailed`, retry com backoff exponencial
5. Envia bind PDU (BindTransmitter/Receiver/Transceiver)
6. Se falha: notifica `BindFailed`, retry
7. Se sucesso: notifica `Connected`, inicia enquireLink e handlePDU
8. Se conexão cai: notifica `Disconnected`, retry

### Workflow de Envio (Submit)

1. Verifica bind (`ErrNotBound`)
2. Verifica window size (`ErrMaxWindowSize`)
3. Registra request no mapa inflight com seq number
4. Aplica rate limiter (se configurado)
5. Serializa e envia PDU
6. Aguarda resposta no channel (com timeout)
7. Valida response PDU ID e status
8. Retorna `ShortMessage` com resposta

### Workflow de EnquireLink

1. Envia EnquireLink periodicamente (intervalo configurável, mín. 10s)
2. Registra timestamp de última resposta recebida
3. Se tempo desde última resposta > `EnquireLinkTimeout` (default 3x intervalo): Unbind + Close + reconnect

### Workflow de Mensagens Longas (Recepção)

1. Recebe DeliverSM com UDH
2. Decodifica UDH header (IEI 0x00 = concatenated short messages)
3. Armazena parte em `MergeHolder` indexado por message ID
4. Quando todas as partes chegam: ordena por part ID, concatena, chama handler
5. Cleanup periódico remove partes expiradas (`MergeInterval`)

## Regras de Domínio

### Modos de Delivery Receipt

- `NoDeliveryReceipt` (0x00) — sem confirmação de entrega
- `FinalDeliveryReceipt` (0x01) — confirmação final
- `FailureDeliveryReceipt` (0x02) — apenas em falha

### Estados de Mensagem (QuerySM)

| Código | Estado | Descrição |
|---|---|---|
| 0 | SCHEDULED | Agendada para envio |
| 1 | ENROUTE | Em trânsito |
| 2 | DELIVERED | Entregue |
| 3 | EXPIRED | Expirada |
| 4 | DELETED | Deletada |
| 5 | UNDELIVERABLE | Não entregável |
| 6 | ACCEPTED | Aceita |
| 7 | UNKNOWN | Desconhecido |
| 8 | REJECTED | Rejeitada |
| 9 | SKIPPED | Ignorada |

### Validade de Mensagem

- Formato SMPP absoluto: `YYMMDDhhmmsstnnp` (spec 7.1.1)
- Calculada como `now().UTC().Add(validity)` com sufixo `"000+"`
- **Implementação:** `smpp/transmitter.go:582-586`

### Tipos de Destino em SubmitMulti

- `0x01` — SME Address (endereço de telefone)
- `0x02` — Distribution List (lista de distribuição)
- **Implementação:** `smpp/transmitter.go:458-474`

---

*Análise de regras de negócio: 2026-03-24*
