# API Specification - go-smpp

**Purpose:** Document the public API surface, method signatures, parameters, error handling, and middleware composition patterns.

---

## Public API Surface

### Transmitter (Send-Only Client)

**Type:** `smpp.Transmitter`

**Configuration Fields:**
```go
type Transmitter struct {
    Addr        string              // SMSC address (e.g., "smsc.example.com:2775")
    User        string              // SystemID for authentication
    Passwd      string              // Password
    SystemType  string              // System type (optional, usually empty)
    EnquireLink time.Duration       // Keepalive interval (default: 10s)
    RespTimeout time.Duration       // Response timeout (default: 1s)
    WindowSize  int                 // Max concurrent requests (0 = unlimited)
    RateLimiter RateLimiter         // Optional rate limiter
    TLS         *tls.Config         // Optional TLS configuration
    Middleware  ConnMiddleware      // Optional connection middleware
}
```

**Methods:**

```go
// Bind connects to SMSC and returns status channel
// Returns: Receive-only channel for connection status updates
func (t *Transmitter) Bind() <-chan ConnStatus

// Submit sends a single SMS message
// Returns: SubmitSMResp with MessageID, or error
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error)

// SubmitLongMsg splits and sends long messages automatically
// Returns: Array of SubmitSMResp (one per part), or error
func (t *Transmitter) SubmitLongMsg(sm *ShortMessage) ([]*SubmitSMResp, error)

// QuerySM queries delivery status of a message
// Parameters: messageID (from SubmitSMResp), sourceAddr (original source)
// Returns: QuerySMResp with message state, or error
func (t *Transmitter) QuerySM(messageID, sourceAddr string) (*QuerySMResp, error)

// Close gracefully closes the connection
// Sends Unbind PDU and waits for response (max 1s timeout)
func (t *Transmitter) Close() error
```

---

### Receiver (Receive-Only Client)

**Type:** `smpp.Receiver`

**Configuration Fields:**
```go
type Receiver struct {
    Addr                 string           // SMSC address
    User                 string           // SystemID
    Passwd               string           // Password
    SystemType           string           // System type (optional)
    EnquireLink          time.Duration    // Keepalive interval (default: 10s)
    Handler              HandlerFunc      // Callback for received PDUs
    TLS                  *tls.Config      // Optional TLS
    LongMessageMerge     bool             // Enable auto-merge (default: true)
    MergeInterval        time.Duration    // Merge check interval (default: 1s)
    MergeCleanupInterval time.Duration    // Cleanup timeout (default: 5min)
}

// HandlerFunc processes received PDUs
type HandlerFunc func(pdu.Body)
```

**Methods:**

```go
// Bind connects to SMSC and starts receiving
func (r *Receiver) Bind() <-chan ConnStatus

// Close stops receiver and closes connection
func (r *Receiver) Close() error
```

---

### Transceiver (Bidirectional Client)

**Type:** `smpp.Transceiver`

**Configuration:** Combines Transmitter and Receiver fields

**Methods:** Combines methods from both Transmitter and Receiver

---

## Data Types

### ShortMessage (SMS to Send)

```go
type ShortMessage struct {
    Src              string          // Source address
    Dst              string          // Destination address
    DstList          []string        // Multiple destinations (SubmitMulti)
    Text             pdutext.Codec   // Message text with encoding
    Validity         time.Duration   // Validity period (optional)
    Register         uint8           // Delivery receipt request (0-2)
    ServiceType      string          // Service type (optional)
    SourceAddrTON    uint8           // Source Type of Number
    SourceAddrNPI    uint8           // Source Numbering Plan
    DestAddrTON      uint8           // Dest Type of Number
    DestAddrNPI      uint8           // Dest Numbering Plan
    ESMClass         uint8           // ESM class
    ProtocolID       uint8           // Protocol ID
    PriorityFlag     uint8           // Priority
    UDH              []byte          // User Data Header (for manual control)
    // Additional fields available via TLV
}
```

### SubmitSMResp (Response from SMSC)

```go
type SubmitSMResp struct {
    MessageID string  // SMSC-assigned message ID
}
```

### QuerySMResp (Message Status Response)

```go
type QuerySMResp struct {
    MessageID    string
    FinalDate    string       // Delivery timestamp
    MessageState MessageState // Current state (0-8)
    ErrorCode    uint8        // Error code if failed
}
```

### ConnStatus (Connection Status)

```go
type ConnStatus interface {
    Status() ConnStatusID
    Error() error
}

type ConnStatusID uint8

const (
    Connected ConnStatusID = 1
    Disconnected
    ConnectionFailed
    BindFailed
)
```

---

## Text Encoding API

### Codec Interface

```go
type Codec interface {
    Type() DataCoding  // Returns data_coding value
    Encode() []byte    // Encodes to bytes
    Decode() []byte    // Decodes from bytes
}
```

### Encoding Constructors

```go
// GSM7 (7-bit, 160 chars)
func pdutext.GSM7(text string) *GSM7

// GSM7 Packed (compressed)
func pdutext.GSM7Packed(text string) *GSM7Packed

// Latin1 (ISO-8859-1, 140 bytes)
func pdutext.Latin1(text string) *Latin1

// UCS2 (Unicode UTF-16BE, 70 chars)
func pdutext.UCS2(text string) *UCS2

// ISO-8859-5 (Cyrillic, 140 bytes)
func pdutext.ISO88595(text string) *ISO88595

// Raw (no encoding, pass bytes directly)
func pdutext.Raw(data interface{}) *Raw  // Accepts string or []byte
```

---

## Error Handling

### Library Errors

```go
var (
    ErrNotConnected  = errors.New("not connected")
    ErrNotBound      = errors.New("not bound")
    ErrTimeout       = errors.New("timeout waiting for response")
    ErrMaxWindowSize = errors.New("max window size reached")
)
```

### SMPP Status Codes

```go
type Status uint32

func (s Status) Error() string

// Common status codes
const (
    ESME_ROK          Status = 0x00000000  // Success
    ESME_RINVMSGLEN   Status = 0x00000001  // Invalid message length
    ESME_RINVSRCADR   Status = 0x0000000A  // Invalid source address
    ESME_RINVDSTADR   Status = 0x0000000B  // Invalid destination
    ESME_RINVPASWD    Status = 0x0000000E  // Invalid password
    ESME_RTHROTTLED   Status = 0x00000058  // Throttled
    // ... 90+ more status codes
)
```

### Error Checking Patterns

```go
resp, err := tx.Submit(sm)
if err != nil {
    // Check for library errors
    if err == smpp.ErrTimeout {
        // Retry
    }
    if err == smpp.ErrNotConnected {
        // Wait for reconnection
    }

    // Check for SMPP status codes
    if status, ok := err.(pdu.Status); ok {
        switch status {
        case pdu.ESME_RTHROTTLED:
            // Backoff and retry
        case pdu.ESME_RINVDSTADR:
            // Invalid destination number
        default:
            // Handle other SMPP errors
        }
    }

    // Network errors
    if errors.Is(err, net.ErrClosed) {
        // Connection closed
    }
}
```

---

## Extension Interfaces

### RateLimiter (Throughput Control)

```go
type RateLimiter interface {
    Wait(ctx context.Context) error
}

// Compatible with golang.org/x/time/rate.Limiter
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 5)  // 10 msg/s, burst of 5
tx := &smpp.Transmitter{
    RateLimiter: limiter,
}
```

### ConnMiddleware (Connection Interception)

```go
type ConnMiddleware func(Conn) Conn

// Example: Logging middleware
func loggingMiddleware(next Conn) Conn {
    return &loggingConn{next: next}
}

tx := &smpp.Transmitter{
    Middleware: loggingMiddleware,
}
```

---

## Testing API

### smpptest Package

```go
import "github.com/devyx-tech/go-smpp/smpp/smpptest"

// Create mock SMSC server
srv := smpptest.NewUnstartedServer(func(c smpp.Conn) {
    // Handle PDUs
    for {
        p, err := c.Read()
        if err != nil {
            return
        }
        // Respond to requests
        if p.Header().ID == pdu.SubmitSMID {
            resp := pdu.NewSubmitSMResp()
            resp.Header().Seq = p.Header().Seq
            c.Write(resp)
        }
    }
})

srv.Start()
defer srv.Close()

// Test with real client
tx := &smpp.Transmitter{Addr: srv.Addr}
```

---

## Usage Examples

### Basic Send

```go
tx := &smpp.Transmitter{
    Addr:   "localhost:2775",
    User:   "test",
    Passwd: "test",
}

conn := tx.Bind()
<-conn  // Wait for connection

resp, err := tx.Submit(&smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: pdutext.Raw("Hello World"),
})

if err != nil {
    log.Fatal(err)
}

log.Printf("MessageID: %s", resp.MessageID)
tx.Close()
```

### Receive with Handler

```go
rx := &smpp.Receiver{
    Addr:   "localhost:2775",
    User:   "test",
    Passwd: "test",
    Handler: func(p pdu.Body) {
        if p.Header().ID == pdu.DeliverSMID {
            fields := p.Fields()
            src := fields[pdufield.SourceAddr]
            text := fields[pdufield.ShortMessage]
            log.Printf("SMS from %s: %s", src, text)
        }
    },
}

conn := rx.Bind()
<-conn
select {}  // Keep running
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 5)  // 10 msg/s

tx := &smpp.Transmitter{
    Addr:        "localhost:2775",
    RateLimiter: limiter,
}
```

### TLS Connection

```go
import "crypto/tls"

tx := &smpp.Transmitter{
    Addr: "smsc.example.com:2776",  // TLS port
    TLS: &tls.Config{
        InsecureSkipVerify: false,
        ServerName:         "smsc.example.com",
    },
}
```

---

## Compatibility

**Go Version:** 1.18+

**SMPP Version:** 3.4 (Protocol Specification Issue 1.2)

**Dependencies:**
- `golang.org/x/text` - Character encoding
- `golang.org/x/time` - Rate limiting (optional, interface only)
- `golang.org/x/net` - Network utilities
- Standard library: `net`, `io`, `context`, `sync`, `time`, `crypto/tls`

---

**Implementation Files:**
- API types: `smpp/*.go`
- PDU types: `smpp/pdu/*.go`
- Text encoding: `smpp/pdu/pdutext/*.go`

**Last Updated:** 2025-10-31
