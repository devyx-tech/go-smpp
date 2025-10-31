# Codebase Navigation Guide - go-smpp

**Purpose:** Help developers and AI assistants quickly navigate and understand the go-smpp codebase structure, key files, data flow, and integration points.

---

## Directory Structure

```
go-smpp/
├── cmd/                          # Command-line tools
│   ├── sms/                      # SMS CLI client
│   │   ├── main.go              # Entry point for sms command
│   │   └── README.md            # CLI documentation
│   └── smsapid/                 # SMS HTTP API daemon
│       ├── main.go              # HTTP server entry point
│       └── README.md            # API documentation
│
├── smpp/                         # Core SMPP library
│   ├── client.go                # Shared client connection logic
│   ├── conn.go                  # Low-level TCP connection wrapper
│   ├── transmitter.go           # Transmitter client (send-only)
│   ├── receiver.go              # Receiver client (receive-only)
│   ├── transceiver.go           # Transceiver client (bidirectional)
│   ├── server.go                # SMPP server (for testing)
│   ├── doc.go                   # Package documentation
│   │
│   ├── pdu/                     # Protocol Data Units
│   │   ├── pdu.go               # PDU base types and interfaces
│   │   ├── header.go            # PDU header (16 bytes)
│   │   ├── body.go              # PDU body interface
│   │   ├── factory.go           # Sequence number factory
│   │   ├── codec.go             # PDU encoding/decoding
│   │   ├── types.go             # Status codes and constants
│   │   ├── submit_sm.go         # SubmitSM PDU
│   │   ├── deliver_sm.go        # DeliverSM PDU
│   │   ├── bind_*.go            # Bind PDU variants
│   │   ├── query_sm.go          # QuerySM PDU
│   │   ├── enquire_link.go      # EnquireLink PDU
│   │   │
│   │   ├── pdufield/            # PDU field definitions
│   │   │   ├── types.go         # Field type definitions
│   │   │   ├── body.go          # Field parsing logic
│   │   │   ├── list.go          # Ordered field list
│   │   │   ├── map.go           # Field key-value map
│   │   │   └── doc.go           # Field documentation
│   │   │
│   │   ├── pdutext/             # Text encoding codecs
│   │   │   ├── codec.go         # Codec interface
│   │   │   ├── gsm7.go          # GSM 7-bit encoding
│   │   │   ├── latin1.go        # ISO-8859-1 encoding
│   │   │   ├── ucs2.go          # Unicode (UTF-16BE) encoding
│   │   │   ├── iso88595.go      # Cyrillic encoding
│   │   │   ├── raw.go           # Raw bytes (no encoding)
│   │   │   └── doc.go           # Encoding documentation
│   │   │
│   │   └── pdutlv/              # Tag-Length-Value fields
│   │       ├── tlv_types.go     # TLV type definitions
│   │       ├── tlv_body.go      # TLV parsing
│   │       ├── tlv_list.go      # TLV list
│   │       ├── tlv_map.go       # TLV key-value map
│   │       └── messagestate.go  # Message state TLV
│   │
│   ├── encoding/                # Character encoding transformations
│   │   └── gsm7.go              # GSM 03.38 encoding implementation
│   │
│   └── smpptest/                # Test utilities
│       ├── server.go            # Mock SMSC server
│       ├── conn.go              # Mock connection
│       └── doc.go               # Test utilities documentation
│
├── docs/                         # User-facing documentation
│   ├── index.md                 # Documentation index
│   ├── stack.md                 # Technology stack
│   ├── patterns.md              # Design patterns
│   ├── features.md              # Feature documentation
│   ├── business-rules.md        # SMPP protocol rules
│   └── integrations.md          # Integration examples
│
├── specs/                        # Technical specifications (this folder)
│   └── technical/               # Technical context documentation
│       ├── index.md             # Documentation index
│       ├── project_charter.md   # Project charter
│       ├── adr/                 # Architectural Decision Records
│       ├── CLAUDE.meta.md       # AI development guide
│       ├── CODEBASE_GUIDE.md    # This file
│       └── ...                  # Other technical docs
│
├── go.mod                        # Go module dependencies
├── go.sum                        # Dependency checksums
├── README.md                     # Project README
├── LICENSE                       # BSD-style license
├── AUTHORS                       # Project authors
├── CONTRIBUTORS                  # Project contributors
└── .travis.yml                   # Travis CI configuration
```

---

## Key Files and Their Roles

### Core Client Files

**`smpp/transmitter.go`** (326 lines)
- **Purpose:** Transmitter client for sending SMS
- **Key Types:** `Transmitter` struct
- **Key Methods:**
  - `Bind() <-chan ConnStatus` - Connect to SMSC
  - `Submit(*ShortMessage) (*SubmitSMResp, error)` - Send single SMS
  - `SubmitLongMsg(*ShortMessage) ([]*SubmitSMResp, error)` - Send long message (auto-split)
  - `QuerySM(msgID, src string) (*QuerySMResp, error)` - Query message status
  - `Close() error` - Close connection
- **Dependencies:** `client.go`, `pdu/submit_sm.go`
- **When to edit:** Adding transmitter features, changing send behavior

**`smpp/receiver.go`** (252 lines)
- **Purpose:** Receiver client for receiving SMS
- **Key Types:** `Receiver` struct, `HandlerFunc` callback
- **Key Methods:**
  - `Bind() <-chan ConnStatus` - Connect and start receiving
  - Internal: `mergeLongMessages()` - Reassemble multi-part messages
  - `Close() error` - Close connection
- **Dependencies:** `client.go`, `pdu/deliver_sm.go`
- **When to edit:** Adding receiver features, changing message handling, adjusting merge logic

**`smpp/transceiver.go`** (289 lines)
- **Purpose:** Transceiver client for bidirectional SMS
- **Key Types:** `Transceiver` struct
- **Key Methods:** Combines methods from Transmitter and Receiver
- **Dependencies:** `client.go`, both `submit_sm.go` and `deliver_sm.go`
- **When to edit:** Adding transceiver-specific features

**`smpp/client.go`** (311 lines)
- **Purpose:** Shared client connection logic
- **Key Types:** `client` struct (internal), `Conn` interface, `ConnStatus` interface
- **Key Functions:**
  - `Bind()` - Reconnection loop with exponential backoff
  - `enquireLink()` - Keepalive management
  - `bind()` - PDU bind function
- **Dependencies:** `conn.go`, `pdu/`
- **When to edit:** Changing reconnection logic, modifying keepalive, altering bind process

**`smpp/conn.go`** (107 lines)
- **Purpose:** Low-level TCP/TLS connection wrapper
- **Key Types:** `conn` struct, `connSwitch` struct
- **Key Functions:**
  - `Dial(addr string, config *tls.Config) (Conn, error)` - Establish TCP/TLS connection
  - `Read()` / `Write()` - PDU I/O
- **Dependencies:** `net`, `crypto/tls`, `pdu/`
- **When to edit:** Adding custom transport, modifying connection handling

---

### PDU (Protocol Data Unit) Files

**`smpp/pdu/pdu.go`** (169 lines)
- **Purpose:** PDU base types and interfaces
- **Key Types:**
  - `Body` interface - Common interface for all PDUs
  - `Header` struct - 16-byte PDU header
  - `Factory` interface - Sequence number generation
- **Key Constants:** PDU command IDs (SubmitSMID, DeliverSMID, etc.)
- **When to edit:** Adding new PDU types, modifying PDU structure

**`smpp/pdu/submit_sm.go`, `deliver_sm.go`, etc.**
- **Purpose:** Individual PDU type implementations
- **Key Types:** PDU-specific structs (SubmitSM, DeliverSM, etc.)
- **Key Methods:** Field list definition, serialization
- **When to edit:** Modifying specific PDU behavior

**`smpp/pdu/codec.go`** (142 lines)
- **Purpose:** PDU encoding/decoding
- **Key Types:** `Codec` struct
- **Key Functions:**
  - `Decode(r io.Reader) (Body, error)` - Deserialize PDU from bytes
  - `Encode(p Body, w io.Writer) error` - Serialize PDU to bytes
- **When to edit:** Changing PDU serialization format

**`smpp/pdu/types.go`** (250 lines)
- **Purpose:** SMPP status codes and type definitions
- **Key Types:** `Status` type
- **Key Constants:** Status codes (ESME_ROK, ESME_RINVPASWD, etc.)
- **When to edit:** Adding new status codes

---

### Text Encoding Files

**`smpp/pdu/pdutext/codec.go`** (85 lines)
- **Purpose:** Text encoding interface and utilities
- **Key Types:** `Codec` interface, `DataCoding` type
- **Key Functions:** Codec selection helpers
- **When to edit:** Adding new encoding types

**`smpp/pdu/pdutext/gsm7.go`** (112 lines)
- **Purpose:** GSM 7-bit encoding
- **Key Types:** `GSM7` struct, `GSM7Packed` struct
- **Dependencies:** `encoding/gsm7.go`
- **When to edit:** Modifying GSM7 encoding behavior

**`smpp/pdu/pdutext/ucs2.go`** (67 lines)
- **Purpose:** Unicode (UTF-16 Big-Endian) encoding
- **Key Types:** `UCS2` struct
- **When to edit:** Modifying Unicode handling

**`smpp/encoding/gsm7.go`** (398 lines)
- **Purpose:** Low-level GSM 03.38 encoding/decoding
- **Key Types:** `Encoder`, `Decoder` (implements `transform.Transformer`)
- **Key Data:** GSM7 character table, escape sequences
- **When to edit:** Fixing GSM7 encoding issues, adding characters

---

### Testing Files

**`smpp/smpptest/server.go`** (187 lines)
- **Purpose:** In-process mock SMSC server for testing
- **Key Types:** `Server` struct
- **Key Functions:**
  - `NewUnstartedServer(handler func(Conn)) *Server`
  - `Start()` - Start listening
  - `Close()` - Shutdown server
- **When to use:** Integration tests requiring mock SMSC

**`smpp/*_test.go`** files
- **Purpose:** Unit and integration tests
- **Naming:** `<file>_test.go` tests `<file>.go`
- **Pattern:** Table-driven tests, example tests
- **When to edit:** Adding tests for new features

---

## Data Flow Patterns

### Outbound Message Flow (Transmitter)

```
Application
    │
    └─ tx.Submit(ShortMessage)
        │
        ├─ Rate limiter check (if configured)
        ├─ Build SubmitSM PDU
        ├─ Assign sequence number
        ├─ Register in inflight map
        ├─ Write PDU to connection
        │   │
        │   └─ conn.Write(pdu.Body)
        │       │
        │       └─ Encode PDU to bytes
        │           │
        │           └─ TCP send to SMSC
        │
        └─ Await SubmitSMResp (or timeout)
            │
            └─ Return MessageID
```

**Files Involved:**
1. `transmitter.go:Submit()` - Entry point
2. `client.go:Write()` - Rate limiting, conn wrapper
3. `pdu/submit_sm.go` - PDU definition
4. `pdu/codec.go:Encode()` - Serialization
5. `conn.go:Write()` - TCP write

---

### Inbound Message Flow (Receiver)

```
SMSC (TCP connection)
    │
    └─ TCP receive bytes
        │
        └─ conn.Read()
            │
            ├─ Decode bytes to PDU
            │   │
            │   └─ pdu.Decode(reader)
            │
            ├─ Check PDU type
            │
            ├─ If DeliverSM:
            │   │
            │   ├─ Check for UDH (long message)
            │   │   │
            │   │   ├─ Yes → Buffer part
            │   │   │   │
            │   │   │   └─ mergeLongMessages() goroutine
            │   │   │       │
            │   │   │       └─ If complete → Reassemble → Handler
            │   │   │
            │   │   └─ No → Handler immediately
            │   │
            │   └─ Send DeliverSMResp (automatic)
            │
            └─ Other PDU types → Process accordingly
```

**Files Involved:**
1. `conn.go:Read()` - TCP read
2. `pdu/codec.go:Decode()` - Deserialization
3. `receiver.go:Bind()` - PDU routing
4. `receiver.go:mergeLongMessages()` - Message assembly
5. Application's `HandlerFunc` - Message processing

---

### Connection Lifecycle

```
Application
    │
    └─ client.Bind()
        │
        ├─ Initialize client struct
        ├─ Start background goroutine
        │   │
        │   └─ Reconnection loop (forever)
        │       │
        │       ├─ Dial TCP/TLS connection
        │       ├─ Send Bind PDU (Transmitter/Receiver/Transceiver)
        │       ├─ Await Bind Response
        │       ├─ Check status code
        │       │   ├─ Success → notify(Connected)
        │       │   └─ Failure → notify(BindFailed) → retry with backoff
        │       │
        │       ├─ Start EnquireLink goroutine
        │       │   │
        │       │   └─ Every EnquireLink interval:
        │       │       ├─ Send EnquireLink PDU
        │       │       ├─ Check last EnquireLinkResp time
        │       │       └─ If timeout → Close connection
        │       │
        │       ├─ PDU read loop
        │       │   │
        │       │   └─ For each PDU:
        │       │       ├─ EnquireLink → Auto-respond
        │       │       ├─ EnquireLinkResp → Update timestamp
        │       │       └─ Other → Route to appropriate handler
        │       │
        │       └─ On error → notify(Disconnected) → retry with backoff
        │
        └─ Return status channel
```

**Files Involved:**
1. `transmitter.go:Bind()` / `receiver.go:Bind()` / `transceiver.go:Bind()`
2. `client.go:Bind()` - Reconnection loop
3. `client.go:enquireLink()` - Keepalive management
4. `conn.go:Read()` / `Write()` - I/O operations

---

## Integration Points

### External Dependencies

**Required Dependencies:**
```go
golang.org/x/text   - Character encoding transformations
golang.org/x/time   - Rate limiting interface (rate.Limiter)
golang.org/x/net    - Network utilities (context integration)
github.com/urfave/cli - CLI framework (cmd/ only)
```

**Standard Library:**
```go
net                 - TCP/TLS connections
io                  - Reader/Writer interfaces
context             - Timeout and cancellation
sync                - Mutexes, atomic operations
time                - Timers, durations
encoding/binary     - Big-endian integer serialization
```

---

### Application Integration Patterns

**Pattern 1: Microservice with HTTP API**
```
HTTP Client → HTTP Server → go-smpp Transmitter → SMSC
                                      ↓
                            Database (persist messages)
```

**Files:** `cmd/smsapid/main.go` - Example implementation

**Pattern 2: Message Queue Consumer**
```
Message Queue → Worker → go-smpp Transmitter → SMSC
(RabbitMQ)                           ↓
                        Database (delivery receipts)
```

**Pattern 3: Bidirectional Service**
```
            ┌──→ go-smpp Transceiver ←──┐
            │                            │
Application Logic ←─────────────────────┘
    ↓
Database / Cache
```

---

## Build and Deployment

### Building the Library

```bash
# Build library (check compilation)
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

### Building CLI Tools

```bash
# Build sms CLI
go build -o sms ./cmd/sms

# Build smsapid daemon
go build -o smsapid ./cmd/smsapid

# Install globally
go install ./cmd/sms
go install ./cmd/smsapid
```

### Container Deployment

```dockerfile
# Example Dockerfile
FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o myapp ./cmd/myapp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/myapp /myapp
CMD ["/myapp"]
```

---

## Debugging Tips

### Enable Verbose Logging

Use `ConnMiddleware` to log PDUs:

```go
func loggingMiddleware(next smpp.Conn) smpp.Conn {
    return &loggingConn{next: next}
}

type loggingConn struct {
    next smpp.Conn
}

func (c *loggingConn) Write(p pdu.Body) error {
    log.Printf("→ Sending: %s (seq=%d)", p.Header().ID, p.Header().Seq)
    return c.next.Write(p)
}

func (c *loggingConn) Read() (pdu.Body, error) {
    p, err := c.next.Read()
    if err == nil {
        log.Printf("← Received: %s (seq=%d)", p.Header().ID, p.Header().Seq)
    }
    return p, err
}

func (c *loggingConn) Close() error {
    return c.next.Close()
}

// Usage
tx := &smpp.Transmitter{
    Middleware: loggingMiddleware,
}
```

### Monitor Connection Status

```go
go func() {
    for status := range tx.Bind() {
        log.Printf("Status: %s, Error: %v", status.Status(), status.Error())
    }
}()
```

### Packet Capture

Use Wireshark/tcpdump to inspect SMPP traffic:

```bash
# Capture SMPP port
tcpdump -i any -s 0 -w smpp.pcap 'port 2775'

# Analyze with Wireshark
wireshark smpp.pcap
```

---

## Common Code Modifications

### Adding a New Text Encoding

1. Create `smpp/pdu/pdutext/myencoding.go`
2. Implement `Codec` interface:
   ```go
   type MyEncoding struct { text string }
   func (m *MyEncoding) Type() DataCoding { return 0x?? }
   func (m *MyEncoding) Encode() []byte { ... }
   func (m *MyEncoding) Decode() []byte { ... }
   ```
3. Add constructor: `func MyEncoding(text string) *MyEncoding`
4. Add tests in `myencoding_test.go`

### Adding a New PDU Type

1. Create `smpp/pdu/my_pdu.go`
2. Define struct embedding `*Codec`
3. Implement `FieldList()` method
4. Add constants for command ID
5. Update `factory.go` if needed
6. Add tests

### Modifying Reconnection Behavior

Edit `smpp/client.go:Bind()`:
- Adjust backoff calculation
- Change max delay
- Add jitter
- Implement circuit breaker

---

**Last Updated:** 2025-10-31
