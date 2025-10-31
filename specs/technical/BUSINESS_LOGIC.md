# Business Logic Documentation - go-smpp

**Purpose:** Document SMPP 3.4 protocol rules, validations, character encoding handling, long message logic, and delivery receipt workflows implemented in the library.

---

## SMPP 3.4 Protocol Rules

### PDU Structure Requirements

**Header Format (16 bytes fixed):**
```
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ Command Length  │  Command ID     │ Command Status  │ Sequence Number │
│    (4 bytes)    │   (4 bytes)     │   (4 bytes)     │   (4 bytes)     │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘
```

**Validation Rules:**
- Command Length ≥ 16 bytes (header size)
- Command Length ≤ 4096 bytes (typical max, SMSC-dependent)
- Command Status = 0x00000000 in requests
- Sequence Number must be unique per connection (1 to 2^32-1)

**Implementation:** `smpp/pdu/header.go`, `smpp/pdu/codec.go`

---

### Bind Operations

**Three Bind Types:**
1. **Bind Transmitter** (0x00000002) - Send SMS only
2. **Bind Receiver** (0x00000001) - Receive SMS only
3. **Bind Transceiver** (0x00000009) - Send and receive

**Required Fields:**
- `system_id`: Max 16 bytes (C-String)
- `password`: Max 9 bytes (C-String)
- `system_type`: Max 13 bytes (optional, usually empty)
- `interface_version`: 0x34 (SMPP 3.4)
- `addr_ton`: Type of Number (usually 0)
- `addr_npi`: Numbering Plan Indicator (usually 0)
- `address_range`: Address range (usually empty)

**Success Response:** `command_status` = 0x00000000
**Failure Responses:** See Status Codes section

**Implementation:** `smpp/client.go:bind()`

---

### SubmitSM (Send SMS)

**Required Fields:**
- `service_type`: Service type (usually empty)
- `source_addr_ton`: Source Type of Number (0-6)
- `source_addr_npi`: Source Numbering Plan (0-18)
- `source_addr`: Source address (max 21 bytes)
- `dest_addr_ton`: Destination Type of Number
- `dest_addr_npi`: Destination Numbering Plan
- `destination_addr`: Destination address (max 21 bytes, **required**)
- `esm_class`: ESM class (default 0x00)
- `protocol_id`: Protocol ID (default 0x00)
- `priority_flag`: Priority (default 0x00)
- `schedule_delivery_time`: Scheduled time (usually empty)
- `validity_period`: Validity period (optional)
- `registered_delivery`: Delivery receipt request (0-2)
- `replace_if_present_flag`: Replace flag (default 0x00)
- `data_coding`: Character encoding (0x00=GSM7, 0x08=UCS2, etc.)
- `sm_default_msg_id`: Default message ID (default 0x00)
- `sm_length`: Message length
- `short_message`: Message payload (max 254 bytes)

**Optional TLV Parameters:**
- `message_payload` (0x0424): For messages > 254 bytes
- `sar_*` fields: For segmented messages
- Many others per SMPP spec

**Response:** `SubmitSMResp` with `message_id` field

**Implementation:** `smpp/transmitter.go:Submit()`, `smpp/pdu/submit_sm.go`

---

### DeliverSM (Receive SMS)

**Key Fields:**
- `service_type`, `source_addr`, `destination_addr`: Same as SubmitSM
- `esm_class`: 0x04 bit set indicates delivery receipt
- `short_message`: Message text or DLR text

**Delivery Receipt Format (in `short_message`):**
```
id:MSGID sub:001 dlvrd:001 submit date:YYMMDDhhmm done date:YYMMDDhhmm stat:DELIVRD err:000 text:...
```

**Auto-Response:** Library automatically sends `DeliverSMResp`

**Implementation:** `smpp/receiver.go`, `smpp/pdu/deliver_sm.go`

---

## Character Encoding Rules

### Supported Encodings

| Encoding | Data Coding | Max Chars | Max Bytes | Use Case |
|----------|-------------|-----------|-----------|----------|
| **GSM7** | 0x00 | 160 (153 with UDH) | 140 | English, basic symbols |
| **GSM7 Packed** | 0x00 | 160 (153 with UDH) | 140 | Compressed GSM7 |
| **Latin1** | 0x03 | 140 | 140 | Western European |
| **UCS2** | 0x08 | 70 (67 with UDH) | 140 | Unicode (emoji, Chinese, Arabic) |
| **ISO-8859-5** | 0x06 | 140 | 140 | Cyrillic |
| **Raw** | Custom | Variable | 140 | Binary data |

### GSM7 Character Set

**Basic Table (128 chars):**
```
@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ ÆæßÉ !"#¤%&'()*+,-./
0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿
abcdefghijklmnopqrstuvwxyzäöñüà
```

**Extended Table (Escape 0x1B + char):**
```
^{}[~]|\€
```

**Implementation:** `smpp/encoding/gsm7.go`, `smpp/pdu/pdutext/gsm7.go`

### Encoding Selection Logic

```go
// Automatic selection
func chooseEncoding(text string) pdutext.Codec {
    if containsOnlyGSM7(text) {
        return pdutext.GSM7(text)  // 160 chars
    }
    if containsOnlyLatin1(text) {
        return pdutext.Latin1(text)  // 140 chars
    }
    return pdutext.UCS2(text)  // 70 chars (supports everything)
}
```

**Rule:** Always match `data_coding` field with actual encoding used

---

## Long Message Concatenation

### UDH (User Data Header) Structure

**Format for 8-bit reference number (IEI 0x00):**
```
┌───────┬───────┬────────────┬─────────────┬──────────────┐
│ IEI   │ IEDL  │ Ref Number │ Total Parts │ Current Part │
│ 0x00  │ 0x03  │  (1 byte)  │  (1 byte)   │  (1 byte)    │
└───────┴───────┴────────────┴─────────────┴──────────────┘
```

**Format for 16-bit reference number (IEI 0x08):**
```
┌───────┬───────┬──────────────────┬─────────────┬──────────────┐
│ IEI   │ IEDL  │ Ref Number (2B)  │ Total Parts │ Current Part │
│ 0x08  │ 0x04  │  (2 bytes)       │  (1 byte)   │  (1 byte)    │
└───────┴───────┴──────────────────┴─────────────┴──────────────┘
```

**Fields:**
- **IEI:** Information Element Identifier
- **IEDL:** IE Data Length (0x03 or 0x04)
- **Ref Number:** Unique message reference (8 or 16 bits)
- **Total Parts:** Total number of parts (1-255)
- **Current Part:** Part number (1 to Total Parts, 1-indexed)

### Character Limits with UDH

| Encoding | Without UDH | With UDH | UDH Size |
|----------|-------------|----------|----------|
| GSM7 | 160 chars | 153 chars | 7 bytes |
| Latin1 | 140 bytes | 133 bytes | 7 bytes |
| UCS2 | 70 chars | 67 chars | 6 bytes |

### Sending Long Messages

**Automatic Splitting (Transmitter):**
```go
// Library automatically splits if message > limit
resp, err := tx.SubmitLongMsg(&smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: pdutext.GSM7(strings.Repeat("A", 200)),
})
// Returns []* SubmitSMResp (one per part)
```

**Algorithm:**
1. Detect message exceeds single SMS limit
2. Calculate part size (153 for GSM7, 67 for UCS2)
3. Generate random 8-bit reference number
4. Split message into parts
5. Add UDH to each part
6. Submit each part as separate SubmitSM
7. Return array of responses

**Implementation:** `smpp/transmitter.go:SubmitLongMsg()`

### Receiving Long Messages

**Automatic Merging (Receiver):**
```go
rx := &smpp.Receiver{
    LongMessageMerge: true,  // Default
    MergeInterval: 1 * time.Second,
    MergeCleanupInterval: 5 * time.Minute,
}
// Handler receives complete message
```

**Algorithm:**
1. Detect UDH in `short_message` (first byte = UDH length)
2. Extract reference number, total parts, current part
3. Store part in memory map: `map[refNum]map[partNum]payload`
4. Timer checks every `MergeInterval` if all parts received
5. If complete (len(parts) == total), reassemble in order
6. Call handler with complete message
7. Cleanup discards incomplete messages after `MergeCleanupInterval`

**Implementation:** `smpp/receiver.go:mergeLongMessages()`

---

## Delivery Receipt Workflows

### Requesting Delivery Receipts

**Field:** `registered_delivery` in SubmitSM

**Values:**
- `0x00`: No delivery receipt
- `0x01`: Delivery receipt requested (final status only)
- `0x02`: Delivery receipt for failed deliveries only

**Example:**
```go
resp, err := tx.Submit(&smpp.ShortMessage{
    Dst:      "5511999999999",
    Text:     pdutext.Raw("Hello"),
    Register: smpp.FinalDeliveryReceipt,  // 0x01
})
// SMSC will send DeliverSM with delivery status later
```

### Receiving Delivery Receipts

**Detection:** `esm_class` bit 2 (0x04) set in DeliverSM indicates DLR

**DLR Format in `short_message`:**
```
id:ABC123 sub:001 dlvrd:001 submit date:2501011200 done date:2501011201 stat:DELIVRD err:000 text:Hello
```

**Fields:**
- `id`: Original message ID from SubmitSMResp
- `sub`: Number of submissions
- `dlvrd`: Number delivered
- `submit date`: Submission timestamp (YYMMDDhhmm)
- `done date`: Delivery timestamp
- `stat`: Message state (DELIVRD, EXPIRED, UNDELIVRD, etc.)
- `err`: Error code
- `text`: First 20 chars of original message

**Parsing Example:**
```go
Handler: func(p pdu.Body) {
    fields := p.Fields()
    esmClass := fields[pdufield.ESMClass].(uint8)

    if esmClass&0x04 != 0 {
        // Delivery receipt
        text := string(fields[pdufield.ShortMessage].([]byte))
        messageID := extractField(text, "id:")
        status := extractField(text, "stat:")

        // Update database with delivery status
        updateDeliveryStatus(messageID, status)
    }
}
```

### Message States

**SMPP Message States:**
- `0` - SCHEDULED (queued)
- `1` - ENROUTE (in transit)
- `2` - DELIVERED (successfully delivered)
- `3` - EXPIRED (validity period expired)
- `4` - DELETED (deleted by SMSC)
- `5` - UNDELIVERABLE (delivery failed)
- `6` - ACCEPTED (accepted by SMSC)
- `7` - UNKNOWN (unknown state)
- `8` - REJECTED (rejected by SMSC)

**Implementation:** `smpp/pdu/pdutlv/messagestate.go`

---

## Validation Rules

### Address Validation

**Format:** Depends on TON (Type of Number) and NPI (Numbering Plan Indicator)

**TON Values:**
- `0` - Unknown
- `1` - International (e.g., +5511999999999)
- `2` - National
- `5` - Alphanumeric (e.g., "MyApp")

**NPI Values:**
- `0` - Unknown
- `1` - ISDN/E.164 (digits only)
- `5` - Private (custom)

**Rules:**
- Max length: 21 bytes
- If NPI=1 (ISDN): digits 0-9 only
- If TON=5 (Alphanumeric): alphanumeric chars allowed

**Implementation:** Field validation in `smpp/pdu/pdufield/types.go`

### Message Length Validation

**Limits:**
- `short_message` field: Max 254 bytes
- With `message_payload` TLV: Up to 64KB (SMSC-dependent)

**Rule:** If message exceeds 254 bytes, use `message_payload` TLV instead of `short_message`

---

## Status Codes

### Common Success/Error Codes

| Code | Constant | Meaning | Action |
|------|----------|---------|--------|
| 0x00000000 | ESME_ROK | Success | None |
| 0x00000001 | ESME_RINVMSGLEN | Invalid message length | Check encoding, message size |
| 0x0000000A | ESME_RINVSRCADR | Invalid source address | Check source format |
| 0x0000000B | ESME_RINVDSTADR | Invalid destination | Check destination format |
| 0x0000000E | ESME_RINVPASWD | Invalid password | Check credentials |
| 0x00000014 | ESME_RSUBMITFAIL | Submit failed | Retry or check SMSC logs |
| 0x00000058 | ESME_RTHROTTLED | Throttled (rate limit) | Backoff and retry |
| 0x00000400 | ESME_RNOROUTE | No route to destination | Check number validity |

**Full List:** `smpp/pdu/types.go` (90+ status codes)

---

## Timeout Rules

### Response Timeouts

**Default Timeout:** 1 second (configurable via `RespTimeout`)

**Applies to:**
- SubmitSM → SubmitSMResp
- QuerySM → QuerySMResp
- Bind → BindResp

**Behavior:** If no response within timeout, return `ErrTimeout`

### EnquireLink Keepalive

**Interval:** 10 seconds (default, configurable via `EnquireLink`)

**Timeout:** 3× EnquireLink interval (30 seconds default)

**Behavior:**
- Library sends EnquireLink PDU every 10 seconds
- If no EnquireLinkResp received in 30 seconds, close connection
- Triggers automatic reconnection

**Implementation:** `smpp/client.go:enquireLink()`

---

## Edge Cases and Special Handling

### Concatenated Messages with Different Encodings

**Rule:** All parts must use same encoding

```go
// Correct: Same encoding for all parts
tx.SubmitLongMsg(&smpp.ShortMessage{
    Text: pdutext.GSM7(longText),  // All parts use GSM7
})

// Wrong: Mixing encodings (library prevents this)
```

### Out-of-Order Part Delivery

**Handled:** Receiver buffers all parts and assembles in correct order

**Implementation:** Parts stored in map keyed by part number, then sorted

### Missing Parts

**Handled:** Cleanup timer discards incomplete messages after 5 minutes (default)

### Duplicate Parts

**Handled:** If same part number arrives twice, last one wins (overwrites)

### Reference Number Collision

**Unlikely:** 8-bit ref number gives 256 unique messages in flight

**If collision:** Messages may merge incorrectly (rare, mitigated by cleanup timer)

---

## SMSC Provider Variations

### Data Coding Differences

**Problem:** Some SMSCs interpret `data_coding=0` differently:
- Some treat as GSM7
- Some treat as ASCII
- Some treat as ISO-8859-1

**Solution:** Test with specific SMSC, adjust encoding strategy

### TON/NPI Requirements

**Problem:** Some SMSCs require specific TON/NPI combinations

**Solution:** Configure per-SMSC:
```go
sm := &smpp.ShortMessage{
    SourceAddrTON: 5,  // Alphanumeric
    SourceAddrNPI: 0,  // Unknown
    DestAddrTON:   1,  // International
    DestAddrNPI:   1,  // ISDN
}
```

### Validity Period Format

**Problem:** SMPP supports absolute and relative time formats; not all SMSCs support both

**Solution:** Test with SMSC, use supported format

---

## Compliance Checklist

✅ PDU header exactly 16 bytes
✅ Sequence numbers unique per connection
✅ Data coding matches text encoding
✅ Address lengths ≤ 21 bytes
✅ Message length ≤ 254 bytes (or use message_payload)
✅ UDH correctly formatted for long messages
✅ EnquireLink sent periodically
✅ DeliverSM auto-responded
✅ Status codes checked on all responses
✅ Bind interface version = 0x34 (SMPP 3.4)

---

**Implementation Files:**
- Business rules: `docs/business-rules.md` (user-facing)
- Protocol implementation: `smpp/pdu/*.go`
- Encoding logic: `smpp/pdu/pdutext/*.go`, `smpp/encoding/gsm7.go`
- Long message handling: `smpp/transmitter.go:SubmitLongMsg()`, `smpp/receiver.go:mergeLongMessages()`

**Last Updated:** 2025-10-31
