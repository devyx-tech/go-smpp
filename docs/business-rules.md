# Regras de Negócio

Este documento descreve as **regras do protocolo SMPP 3.4** implementadas na biblioteca **go-smpp**, incluindo validações, restrições e comportamentos obrigatórios.

---

## Regras Críticas do Protocolo SMPP

### 1. Estrutura de PDU (Protocol Data Unit)

**Descrição**: Toda comunicação SMPP é feita via PDUs com estrutura binária padronizada.

**Justificativa**: Requisito fundamental do protocolo SMPP 3.4

**Implementação**: [smpp/pdu/pdu.go:60-75](../smpp/pdu/pdu.go#L60-L75)

#### Regra: PDU Header (16 bytes fixos)

**Campos obrigatórios** (ordem e tamanho fixos):

| Campo | Bytes | Tipo | Descrição |
|-------|-------|------|-----------|
| Command Length | 4 | uint32 | Tamanho total do PDU (header + body) |
| Command ID | 4 | uint32 | Tipo de PDU (ex: 0x00000004 = SubmitSM) |
| Command Status | 4 | uint32 | Status de resposta (0x00000000 = OK) |
| Sequence Number | 4 | uint32 | Número sequencial para correlação request/response |

**Validações**:
```go
// Tamanho mínimo: 16 bytes (header)
if len(data) < 16 {
    return nil, errors.New("PDU too short")
}

// Tamanho máximo: 4096 bytes (prevenção de DoS)
if commandLength > 4096 {
    return nil, errors.New("PDU too large")
}

// Command Status deve ser 0 em requests
if isRequest && status != 0 {
    return nil, errors.New("invalid status in request")
}
```

**Localização**: [pdu/pdu.go:138-152](../smpp/pdu/pdu.go#L138-L152)

---

### 2. Binding (Autenticação)

**Descrição**: Cliente deve autenticar-se com SMSC antes de enviar/receber mensagens.

**Justificativa**: Segurança e identificação de cliente

**Implementação**: [client.go:89-132](../smpp/client.go#L89-L132)

#### Regra: Tipos de Bind

| Tipo | Command ID | Propósito |
|------|-----------|-----------|
| Bind Transmitter | 0x00000002 | Apenas envio de SMS |
| Bind Receiver | 0x00000001 | Apenas recebimento de SMS |
| Bind Transceiver | 0x00000009 | Envio e recebimento simultâneo |

**Validações**:
```go
// SystemID é obrigatório (max 16 bytes)
if len(systemID) == 0 || len(systemID) > 16 {
    return errors.New("invalid system_id")
}

// Password é obrigatório (max 9 bytes)
if len(password) == 0 || len(password) > 9 {
    return errors.New("invalid password")
}

// Interface Version deve ser 0x34 (SMPP 3.4)
interfaceVersion := 0x34
```

#### Regra: Bind Response

**Status Code 0x00000000**: Autenticação bem-sucedida

**Status Code != 0**: Falha de autenticação

```go
if resp.Header().Status != 0 {
    return fmt.Errorf("bind failed: %s", resp.Header().Status)
}
```

**Exceções**: Após bind failure, conexão deve ser fechada e reconectada

**Localização**: [client.go:113-129](../smpp/client.go#L113-L129)

---

### 3. Sequence Numbers

**Descrição**: Cada PDU deve ter número sequencial único para correlação request/response.

**Justificativa**: Permite múltiplas requisições concorrentes sem ambiguidade

**Implementação**: [client.go:78-82](../smpp/client.go#L78-L82)

#### Regra: Geração de Sequence Numbers

**Requisitos**:
- Deve ser **auto-incrementado** (1, 2, 3, ...)
- Deve ser **único por conexão**
- Deve ser **thread-safe** (múltiplas goroutines)
- Deve **reiniciar em 1** após reconexão

**Implementação**:
```go
type factory struct {
    seq uint32
}

func (f *factory) NewSeq() uint32 {
    return atomic.AddUint32(&f.seq, 1)  // Thread-safe increment
}
```

#### Regra: Correlação Request/Response

**Requisito**: Response PDU deve ter **mesmo sequence number** que request

**Implementação**: [transmitter.go:189-236](../smpp/transmitter.go#L189-L236)

```go
// Armazenar request em map
inflight := make(map[uint32]chan *tx)
seq := p.Header().Seq
inflight[seq] = responseChannel

// Ao receber response, correlacionar por seq
if ch, ok := inflight[resp.Header().Seq]; ok {
    ch <- resp
    delete(inflight, resp.Header().Seq)
}
```

**Exceções**:
- **EnquireLink**: Pode ter sequence number qualquer, não exige correlação estrita
- **GenericNack**: Response de erro genérico, sequence number indica request que falhou

---

### 4. EnquireLink Keepalive

**Descrição**: Heartbeat periódico para manter conexão viva e detectar falhas.

**Justificativa**: Conexões TCP podem cair silenciosamente; keepalive detecta falhas

**Implementação**: [client.go:134-162](../smpp/client.go#L134-L162)

#### Regra: Intervalo de EnquireLink

**Padrão**: 10 segundos (configurável)

```go
tx := &smpp.Transmitter{
    EnquireLink: 10 * time.Second,  // Padrão
}
```

**Timeout de Resposta**: 3× intervalo de EnquireLink

```go
timeout := 3 * t.EnquireLink  // 30 segundos se EnquireLink = 10s
```

#### Regra: Detecção de Falha

**Condição**: Se EnquireLinkResp não chegar em 3× intervalo

**Ação**: Considerar conexão morta e reconectar

```go
select {
case <-time.After(timeout):
    // Conexão morta - fechar e reconectar
    conn.Close()
    return
}
```

**Localização**: [client.go:146-156](../smpp/client.go#L146-L156)

---

### 5. Reconexão Automática

**Descrição**: Em caso de falha de conexão, cliente deve reconectar automaticamente.

**Justificativa**: Resiliência; conexões SMPP frequentemente caem

**Implementação**: [transmitter.go:137-175](../smpp/transmitter.go#L137-L175)

#### Regra: Exponential Backoff

**Algoritmo**:
```
delay = min(initialDelay × 2^attempts, maxDelay)
```

**Valores**:
- `initialDelay`: 1 segundo
- `maxDelay`: 120 segundos (2 minutos)

**Implementação**:
```go
delay := time.Second  // Inicial: 1s

for {
    err := connect()
    if err == nil {
        delay = time.Second  // Reset ao conectar
        break
    }

    time.Sleep(delay)
    delay *= 2                      // Dobrar: 2s, 4s, 8s, ...
    if delay > 120*time.Second {
        delay = 120 * time.Second   // Cap em 2min
    }
}
```

**Exceções**:
- Se `Close()` for chamado explicitamente, parar reconexão
- Se bind falhar por credenciais inválidas, ainda assim continuar tentando (pode ser erro temporário do SMSC)

**Localização**: [transmitter.go:151-168](../smpp/transmitter.go#L151-L168)

---

### 6. Submissão de Mensagens (SubmitSM)

**Descrição**: Regras para envio de SMS via SubmitSM PDU.

**Implementação**: [transmitter.go:240-335](../smpp/transmitter.go#L240-L335)

#### Regra: Campos Obrigatórios

| Campo | Tipo | Validação |
|-------|------|-----------|
| source_addr | C-String | Max 21 bytes |
| destination_addr | C-String | Max 21 bytes, obrigatório |
| short_message | Octet String | Max 254 bytes (ou 140 se data_coding = GSM7) |
| data_coding | uint8 | Deve corresponder ao encoding real |

**Validações**:
```go
// Destino é obrigatório
if len(sm.Dst) == 0 {
    return nil, errors.New("destination address required")
}

// Source pode ser vazio (SMSC usa padrão)
// mas se especificado, max 21 bytes
if len(sm.Src) > 21 {
    return nil, errors.New("source address too long")
}
```

**Localização**: [transmitter.go:282-308](../smpp/transmitter.go#L282-L308)

#### Regra: Data Coding vs Text Encoding

**Requisito**: `data_coding` field deve corresponder ao encoding do texto

| Text Type | Data Coding | Max Bytes |
|-----------|-------------|-----------|
| GSM7 | 0x00 | 160 chars (septets) |
| Latin1 | 0x03 | 140 bytes |
| UCS2 | 0x08 | 140 bytes (70 chars UTF-16) |
| ISO-8859-5 | 0x06 | 140 bytes |
| Raw/Binary | 0x04 | 140 bytes |

**Implementação**: Codecs automaticamente definem data_coding correto

```go
type Codec interface {
    Type() DataCoding  // Retorna data_coding apropriado
    Encode() []byte
    Decode() []byte
}

// Uso
text := pdutext.UCS2("你好")
// text.Type() retorna 0x08 (UCS2)
```

**Localização**: [pdutext/codec.go](../smpp/pdu/pdutext/codec.go)

#### Regra: Validity Period

**Formato**: SMPP Absolute ou Relative Time

**Absoluto**: `YYMMDDhhmmsstnnp`
- `YY`: Ano (2 dígitos)
- `MM`: Mês
- `DD`: Dia
- `hh`: Hora
- `mm`: Minuto
- `ss`: Segundo
- `t`: Décimos de segundo (0-9)
- `nn`: Timezone offset (quarters de hora)
- `p`: Timezone sign ('+' ou '-')

**Relativo**: `YYMMDDhhmmss000R`
- `R`: Indica formato relativo

**Implementação**: [pdu/pdufield/field.go:140-165](../smpp/pdu/pdufield/field.go#L140-L165)

```go
// Converter time.Duration para SMPP relative time
validity := 24 * time.Hour  // 24 horas
smppTime := toSMPPTime(validity)  // "000001000000000R"
```

---

### 7. Múltiplos Destinos (SubmitMulti)

**Descrição**: Envio de mesma mensagem para múltiplos destinos em uma requisição.

**Implementação**: [pdu/submit_multi.go](../smpp/pdu/submit_multi.go)

#### Regra: Limite de Destinos

**Máximo**: 254 destinos por SubmitMulti

**Validação**:
```go
if len(destinations) > 254 {
    return errors.New("too many destinations (max 254)")
}

if len(destinations) == 0 {
    return errors.New("at least one destination required")
}
```

**Justificativa**: Limite do protocolo SMPP 3.4

#### Regra: Tipos de Destino

**SME Address**: Endereço individual
```go
type SMEAddress struct {
    DestAddrTON  uint8   // Type of Number
    DestAddrNPI  uint8   // Numbering Plan Indicator
    DestAddr     string  // Endereço
}
```

**Distribution List**: Lista de distribuição nomeada (raro)
```go
type DistributionList struct {
    Name string
}
```

**Resposta**: SubmitMultiResp contém lista de sucesso/falha por destino

**Localização**: [pdu/submit_multi.go:40-85](../smpp/pdu/submit_multi.go)

---

### 8. Mensagens Longas (UDH - User Data Header)

**Descrição**: Mensagens que excedem limite de SMS único devem ser divididas com UDH.

**Implementação**: [transmitter.go:372-450](../smpp/transmitter.go#L372-L450), [receiver.go:145-220](../smpp/receiver.go#L145-L220)

#### Regra: Limites de Mensagem

| Encoding | Sem UDH | Com UDH | Bytes UDH |
|----------|---------|---------|-----------|
| GSM7 | 160 chars | 153 chars | 7 bytes |
| Latin1 | 140 bytes | 133 bytes | 7 bytes |
| UCS2 | 140 bytes (70 chars) | 134 bytes (67 chars) | 6 bytes |

**Justificativa**: UDH ocupa espaço do payload

#### Regra: Estrutura UDH para Concatenação

**Formato IEI 0x00** (3-byte reference):
```
[0x00] [0x03] [RefNum] [TotalParts] [CurrentPart]
```

**Formato IEI 0x08** (2-byte reference) - alternativo:
```
[0x08] [0x04] [RefNumHi] [RefNumLo] [TotalParts] [CurrentPart]
```

**Campos**:
- **IEI** (Information Element Identifier): 0x00 ou 0x08
- **IEDL** (IE Data Length): 0x03 (3 bytes) ou 0x04 (4 bytes)
- **RefNum**: Número de referência único (8 ou 16 bits)
- **TotalParts**: Total de partes (1-255)
- **CurrentPart**: Parte atual (1-indexed, 1 a TotalParts)

**Validações**:
```go
// Número de partes máximo
if totalParts > 255 {
    return errors.New("too many message parts (max 255)")
}

// Current part deve estar dentro do range
if currentPart < 1 || currentPart > totalParts {
    return errors.New("invalid current part number")
}

// RefNum deve ser consistente entre partes
if part.RefNum != expectedRefNum {
    return errors.New("mismatched reference number")
}
```

**Localização**: [transmitter.go:391-418](../smpp/transmitter.go#L391-L418)

#### Regra: Merge de Mensagens Longas (Receiver)

**Algoritmo**:
1. Detectar UDH em `short_message` field (primeiro byte = 0x05 indica UDH presente)
2. Extrair IEI, RefNum, TotalParts, CurrentPart
3. Armazenar em map: `map[RefNum]map[CurrentPart]MessagePart`
4. Timer periódico verifica se `len(parts) == TotalParts`
5. Se completo, ordenar partes (1, 2, 3, ...) e concatenar payloads
6. Chamar Handler com mensagem completa
7. Cleanup: remover partes incompletas após `MergeCleanupInterval`

**Configuração**:
```go
rx := &smpp.Receiver{
    LongMessageMerge:     true,              // Habilitar merge
    MergeInterval:        1 * time.Second,   // Verificar a cada 1s
    MergeCleanupInterval: 5 * time.Minute,   // Expirar após 5min
}
```

**Exceções**:
- Se partes não chegarem completas em `MergeCleanupInterval`, descartar
- Se `LongMessageMerge = false`, entregar cada parte individualmente ao Handler

**Localização**: [receiver.go:145-220](../smpp/receiver.go#L145-L220)

---

### 9. Delivery Receipts (DLR)

**Descrição**: Notificações de entrega de mensagem enviada.

**Implementação**: [pdu/pdufield/defs.go:189-197](../smpp/pdu/pdufield/defs.go#L189-L197)

#### Regra: Tipos de Delivery Receipt

| Valor | Enum | Descrição |
|-------|------|-----------|
| 0 | NoDeliveryReceipt | Sem recibo |
| 1 | FinalDeliveryReceipt | Apenas recibo final (entregue ou falhou) |
| 2 | FailureDeliveryReceipt | Apenas em caso de falha |

**Campo PDU**: `registered_delivery` (uint8)

**Configuração**:
```go
sm := &smpp.ShortMessage{
    Src:      "1234",
    Dst:      "5511999999999",
    Text:     pdutext.Raw("Hello"),
    Register: smpp.FinalDeliveryReceipt,  // Solicitar DLR
}
```

#### Regra: Recebimento de DLR

**Formato**: DLR chega via **DeliverSM** PDU com flag especial

**Campo**: `esm_class` bit 2 = 1 indica DLR

**Parsing**:
```go
Handler: func(p pdu.Body) {
    if p.Header().ID == pdu.DeliverSMID {
        fields := p.Fields()
        esmClass := fields[pdufield.ESMClass].(uint8)

        if esmClass & 0x04 != 0 {  // Bit 2 set = DLR
            // Processar DLR
            messageID := fields[pdufield.ReceiptedMessageID]
            finalStatus := fields[pdufield.MessageState]
        }
    }
}
```

**Localização**: [pdu/deliver_sm.go](../smpp/pdu/deliver_sm.go)

---

### 10. Query de Status de Mensagem

**Descrição**: Consultar status de entrega de mensagem enviada.

**Implementação**: [transmitter.go:337-370](../smpp/transmitter.go#L337-L370)

#### Regra: Campos Obrigatórios

| Campo | Descrição | Validação |
|-------|-----------|-----------|
| message_id | ID retornado em SubmitSMResp | Obrigatório, max 65 bytes |
| source_addr | Endereço origem original | Deve ser igual ao da submissão |

**Uso**:
```go
resp, err := tx.QuerySM(messageID, sourceAddr)
```

#### Regra: Estados de Mensagem

**Enum**: `MessageState` (uint8)

| Valor | Estado | Significado |
|-------|--------|-------------|
| 0 | SCHEDULED | Agendada para envio |
| 1 | ENROUTE | Em rota para destino |
| 2 | DELIVERED | Entregue com sucesso |
| 3 | EXPIRED | Expirou antes de entrega |
| 4 | DELETED | Deletada pelo SMSC |
| 5 | UNDELIVERABLE | Não entregável (número inválido, etc.) |
| 6 | ACCEPTED | Aceita pelo SMSC |
| 7 | UNKNOWN | Estado desconhecido |
| 8 | REJECTED | Rejeitada pelo SMSC |

**Resposta**:
```go
type QuerySMResp struct {
    MessageID    string
    FinalDate    string       // Data de finalização (se aplicável)
    MessageState MessageState // Estado atual
    ErrorCode    uint8        // Código de erro (se falhou)
}
```

**Localização**: [pdu/query_sm.go](../smpp/pdu/query_sm.go)

---

## Validações e Restrições

### Validações de Campos

#### 1. Endereços (Source/Destination)

**Tipo**: C-String (null-terminated)

**Validações**:
```go
// Comprimento máximo
if len(addr) > 21 {
    return errors.New("address too long (max 21)")
}

// Caracteres válidos (dependendo de NPI)
// NPI = 1 (ISDN): apenas dígitos 0-9
// NPI = 5 (Private): alfanumérico permitido
```

#### 2. SystemID e Password

**SystemID**:
- Tipo: C-String
- Max: 16 bytes
- Obrigatório para bind

**Password**:
- Tipo: C-String
- Max: 9 bytes
- Obrigatório para bind

**Localização**: [pdu/bind_transmitter.go](../smpp/pdu/bind_transmitter.go)

#### 3. Short Message

**Tamanho**: Max 254 bytes (limite do campo `sm_length`)

**Validação**:
```go
if len(shortMessage) > 254 {
    return errors.New("short_message too long (max 254 bytes)")
}
```

**Exceção**: Se usar `message_payload` TLV, limite sobe para 64KB

#### 4. TON (Type of Number)

**Valores Válidos**:
```go
const (
    TON_Unknown       = 0  // Desconhecido
    TON_International = 1  // Internacional (ex: +5511999999999)
    TON_National      = 2  // Nacional (ex: 011999999999)
    TON_NetworkSpecific = 3  // Específico da rede
    TON_SubscriberNumber = 4  // Número de assinante
    TON_Alphanumeric  = 5  // Alfanumérico (ex: "MyApp")
    TON_Abbreviated   = 6  // Abreviado
)
```

#### 5. NPI (Numbering Plan Indicator)

**Valores Válidos**:
```go
const (
    NPI_Unknown = 0   // Desconhecido
    NPI_ISDN    = 1   // E.164 (internacional)
    NPI_Data    = 3   // X.121
    NPI_Telex   = 4   // F.69
    NPI_Private = 5   // Privado
)
```

---

## Políticas e Workflows

### Workflow de Conexão (Transmitter)

```
1. Dial TCP → SMSC (porta 2775 ou 2776)
   ↓
2. Enviar BindTransmitter PDU
   ↓
3. Aguardar BindTransmitterResp
   ↓
4a. Status = 0 → Conectado
    ↓
    Iniciar EnquireLink timer
    ↓
    Pronto para Submit

4b. Status != 0 → Falha de Bind
    ↓
    Fechar conexão
    ↓
    Aguardar backoff
    ↓
    Voltar ao passo 1
```

**Localização**: [client.go:89-132](../smpp/client.go#L89-L132)

### Workflow de Envio (SubmitSM)

```
1. Verificar se está conectado (bound)
   ↓
2. Se WindowSize > 0, verificar limite de requests concorrentes
   ↓
3. Se RateLimiter configurado, aguardar permissão
   ↓
4. Criar SubmitSM PDU com sequence number
   ↓
5. Armazenar em inflight map
   ↓
6. Escrever PDU na conexão
   ↓
7. Aguardar SubmitSMResp (timeout = RespTimeout)
   ↓
8a. Response recebida → Retornar MessageID
8b. Timeout → Retornar ErrTimeout
8c. Erro de rede → Reconectar, retornar erro
```

**Localização**: [transmitter.go:240-335](../smpp/transmitter.go#L240-L335)

### Workflow de Recebimento (DeliverSM)

```
1. Loop infinito aguardando PDUs
   ↓
2. Read PDU da conexão
   ↓
3. Verificar tipo de PDU
   ↓
4a. DeliverSM:
    ↓
    Se LongMessageMerge habilitado:
      ↓
      Verificar UDH
      ↓
      Se mensagem longa:
        - Armazenar parte
        - Aguardar outras partes
        - Se completo, montar e chamar Handler
      ↓
      Se mensagem única:
        - Chamar Handler diretamente
    ↓
    Enviar DeliverSMResp automaticamente

4b. EnquireLinkResp:
    - Resetar timer de keepalive

4c. Outros PDUs:
    - Chamar Handler (se configurado)
```

**Localização**: [receiver.go:117-143](../smpp/receiver.go#L117-L143)

---

## Tratamento de Erros

### Códigos de Status SMPP

**Localização**: [pdu/pdu.go:154-250](../smpp/pdu/pdu.go#L154-L250)

#### Erros Comuns

| Código | Nome | Significado | Ação Recomendada |
|--------|------|-------------|------------------|
| 0x00000000 | ESME_ROK | Sucesso | Nenhuma |
| 0x00000001 | ESME_RINVMSGLEN | Tamanho de mensagem inválido | Verificar encoding e tamanho |
| 0x00000002 | ESME_RINVCMDLEN | Tamanho de comando inválido | Verificar construção de PDU |
| 0x00000003 | ESME_RINVCMDID | Command ID inválido | Verificar tipo de PDU |
| 0x00000004 | ESME_RINVBNDSTS | Bind status incorreto | Re-bind |
| 0x0000000A | ESME_RINVSRCADR | Endereço origem inválido | Verificar formato de número |
| 0x0000000B | ESME_RINVDSTADR | Endereço destino inválido | Verificar formato de número |
| 0x0000000E | ESME_RINVPASWD | Password inválido | Corrigir credenciais |
| 0x00000014 | ESME_RSUBMITFAIL | Submissão falhou | Retry com backoff |
| 0x00000033 | ESME_RINVDLNAME | Distribution list inválido | Verificar nome da lista |
| 0x00000045 | ESME_RSUBMITFAIL | Erro de submissão | Verificar parâmetros |
| 0x00000058 | ESME_RTHROTTLED | Throttling (rate limit) | Aguardar e retry |

**Implementação**:
```go
type Status uint32

const (
    ESME_ROK          Status = 0x00000000
    ESME_RINVMSGLEN   Status = 0x00000001
    ESME_RINVCMDLEN   Status = 0x00000002
    // ...
)

func (s Status) Error() string {
    return fmt.Sprintf("SMPP Status: 0x%08x", uint32(s))
}
```

### Erros da Biblioteca

**Localização**: [transmitter.go:27-30](../smpp/transmitter.go#L27-L30)

```go
var (
    ErrNotConnected  = errors.New("not connected")
    ErrNotBound      = errors.New("not bound")
    ErrTimeout       = errors.New("timeout waiting for response")
    ErrMaxWindowSize = errors.New("max window size reached")
)
```

**Tratamento**:
```go
_, err := tx.Submit(sm)
if err != nil {
    switch {
    case err == smpp.ErrNotConnected:
        // Aguardar reconexão
        <-tx.Bind()

    case err == smpp.ErrTimeout:
        // Retry
        tx.Submit(sm)

    case errors.Is(err, net.ErrClosed):
        // Conexão fechada - aguardar reconexão

    default:
        if status, ok := err.(pdu.Status); ok {
            // Erro SMPP - verificar código
            if status == pdu.ESME_RTHROTTLED {
                time.Sleep(1 * time.Second)  // Backoff
            }
        }
    }
}
```

---

## Compliance e Padrões

### SMPP 3.4 Specification

**Referência**: [SMPP v3.4 Specification](http://www.smsforum.net/)

**Áreas de Conformidade**:
- ✅ PDU Structure (Header + Body)
- ✅ Bind/Unbind operations
- ✅ SubmitSM / SubmitMulti
- ✅ DeliverSM
- ✅ QuerySM
- ✅ EnquireLink keepalive
- ✅ Status codes completos
- ✅ TLV fields opcionais
- ⚠️ DataSM (não implementado)
- ⚠️ CancelSM / ReplaceSM (não implementados)
- ⚠️ AlertNotification (não implementado)

### Character Encodings

**GSM 03.38**: ✅ Completamente implementado
- Alphabet básico (128 caracteres)
- Escape sequences para caracteres estendidos
- Packed format (7 bits)

**ISO-8859-1 (Latin1)**: ✅ Implementado via golang.org/x/text

**UCS2 (UTF-16 BE)**: ✅ Implementado

**ISO-8859-5 (Cyrillic)**: ✅ Implementado

---

## Regras Específicas de Implementação

### 1. Thread Safety

**Regra**: Todas as operações públicas devem ser thread-safe

**Implementação**:
- `sync.RWMutex` para proteção de estado compartilhado
- `atomic` operations para sequence numbers
- Channels para comunicação entre goroutines

**Localização**: [client.go:194-215](../smpp/client.go#L194-L215)

### 2. Graceful Shutdown

**Regra**: `Close()` deve drenar inflight requests antes de fechar

**Implementação**:
```go
func (t *Transmitter) Close() error {
    t.mu.Lock()
    defer t.mu.Unlock()

    // Aguardar inflight requests (com timeout)
    done := make(chan struct{})
    go func() {
        // Aguardar todos requests completarem
        close(done)
    }()

    select {
    case <-done:
    case <-time.After(5 * time.Second):
        // Timeout - fechar mesmo com requests pendentes
    }

    return t.conn.Close()
}
```

### 3. Context Propagation

**Regra**: Usar `context.Context` para cancelamento e timeouts

**Implementação**:
```go
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error) {
    ctx := context.Background()
    if t.RespTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, t.RespTimeout)
        defer cancel()
    }

    // Usar ctx em operações
    return t.submitWithContext(ctx, sm)
}
```

**Localização**: [transmitter.go:252-257](../smpp/transmitter.go#L252-L257)

---

## Resumo de Regras Críticas

| Regra | Implementação | Exceções |
|-------|--------------|----------|
| PDU Header 16 bytes fixos | pdu.go:60-75 | Nenhuma |
| Bind obrigatório antes de Submit | client.go:89-132 | Nenhuma |
| Sequence numbers únicos | client.go:78-82 | Reinicia após reconexão |
| EnquireLink a cada 10s | client.go:134-162 | Configurável |
| Reconexão com exponential backoff | transmitter.go:151-168 | Max 120s |
| Destino obrigatório em SubmitSM | transmitter.go:282-308 | Nenhuma |
| Max 254 destinos em SubmitMulti | pdu/submit_multi.go | Limite do protocolo |
| UDH para mensagens longas | transmitter.go:391-418 | Opcional |
| Data coding deve corresponder encoding | pdutext/codec.go | Auto-gerenciado por Codec |
| Status 0 = sucesso | pdu.go:154-250 | Nenhuma |

---

## Próximos Passos

Consulte:
- [**Funcionalidades**](features.md) - Como usar cada feature
- [**Integração**](integrations.md) - Integrar em seus projetos
- [**Stack**](stack.md) - Arquitetura e dependências
- [**Padrões**](patterns.md) - Design patterns
