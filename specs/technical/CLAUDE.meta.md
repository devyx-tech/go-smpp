# AI Development Guide - go-smpp

**Target Audience:** AI assistants (Claude, GPT, etc.) helping developers work with go-smpp

**Purpose:** Provide code style patterns, testing approaches, common patterns, and SMPP-specific gotchas extracted from the codebase

---

## Code Style Patterns

### Naming Conventions

**Packages:**
```go
// Lowercase, singular
package smpp
package pdu
package pdufield
package pdutext
package pdutlv
package encoding
```

**Types:**
```go
// PascalCase for exported types
type Transmitter struct { ... }
type BindTransmitter struct { ... }
type ConnStatus interface { ... }

// camelCase for internal types
type client struct { ... }
type connSwitch struct { ... }
```

**Functions and Methods:**
```go
// PascalCase for exported
func (t *Transmitter) Bind() <-chan ConnStatus
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error)

// camelCase for internal
func (c *client) notify(ev ConnStatus)
func dial(addr string, config *tls.Config) (Conn, error)

// NO "Get" prefix for getters
func (c ConnStatus) Status() ConnStatusID  // NOT GetStatus()
func (p *PDU) Header() *Header             // NOT GetHeader()
```

**Constants:**
```go
// PascalCase, group related constants
const (
    Connected ConnStatusID = iota + 1
    Disconnected
    ConnectionFailed
    BindFailed
)

// Protocol constants with descriptive names
const (
    SubmitSMID         = 0x00000004
    SubmitSMRespID     = 0x80000004
    BindTransmitterID  = 0x00000002
)
```

**Variables:**
```go
// camelCase
var inflight map[uint32]chan *tx
var multipartMessages map[uint8]*multipartMessage

// Package-level errors with "Err" prefix
var (
    ErrNotConnected  = errors.New("not connected")
    ErrTimeout       = errors.New("timeout waiting for response")
)
```

---

## File Organization Patterns

### File Naming

```
// Pattern: lowercase with underscores
transmitter.go          // Type definition and main methods
transmitter_test.go     // Tests for transmitter

// Protocol types named after PDU
submit_sm.go            // SubmitSM PDU
deliver_sm.go           // DeliverSM PDU
bind_transmitter.go     // BindTransmitter PDU

// Test files parallel implementation files
conn.go → conn_test.go
encoding/gsm7.go → encoding/gsm7_test.go
```

### File Structure

```go
// 1. Package declaration
package smpp

// 2. Imports (grouped and sorted)
import (
    "context"
    "errors"
    "io"
    "sync"
    "time"

    "golang.org/x/net/context"

    "github.com/devyx-tech/go-smpp/smpp/pdu"
    "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"
)

// 3. Package-level constants
const (
    DefaultEnquireLink = 10 * time.Second
)

// 4. Package-level variables (especially errors)
var (
    ErrNotConnected = errors.New("not connected")
)

// 5. Type definitions
type Transmitter struct { ... }

// 6. Constructor functions (if any)
func NewTransmitter(...) *Transmitter { ... }

// 7. Methods grouped by type
func (t *Transmitter) Bind() <-chan ConnStatus { ... }
func (t *Transmitter) Submit(...) { ... }
func (t *Transmitter) Close() error { ... }

// 8. Helper functions
func dial(addr string) (net.Conn, error) { ... }
```

---

## Common Patterns

### Pattern 1: Channel-Based Async Communication

```go
// Return receive-only channel for status notifications
func (t *Transmitter) Bind() <-chan ConnStatus {
    c := &client{
        Status: make(chan ConnStatus, 10),  // Buffered
    }
    go c.Bind()  // Background goroutine
    return c.Status
}

// Consumer can choose blocking or non-blocking
status := <-conn         // Block until status
select {
case status := <-conn:   // Non-blocking with timeout
case <-time.After(5*time.Second):
}
```

**When to use:** Async event notification, status updates, continuous monitoring

### Pattern 2: Context for Timeouts

```go
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error) {
    ctx := context.Background()
    if t.RespTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, t.RespTimeout)
        defer cancel()
    }

    select {
    case resp := <-respChan:
        return resp, nil
    case <-ctx.Done():
        return nil, ErrTimeout
    }
}
```

**When to use:** Operations with timeouts, cancellation support

### Pattern 3: Sync.Once for One-Time Initialization

```go
type Transmitter struct {
    once sync.Once
    conn *connection
}

func (t *Transmitter) Close() error {
    t.once.Do(func() {
        close(t.stopChan)
        t.conn.Close()
    })
    return nil
}
```

**When to use:** Ensuring Close() is idempotent, one-time setup

### Pattern 4: Interface-Based Extension Points

```go
// Define small interfaces
type Conn interface {
    Read() (pdu.Body, error)
    Write(pdu.Body) error
    Close() error
}

// Accept interface, return struct
func dial(addr string) Conn { ... }                    // Returns struct
func wrap(c Conn) Conn { return &loggingConn{c} }     // Accepts interface
```

**When to use:** Testability, middleware, custom implementations

### Pattern 5: Table-Driven Tests

```go
func TestCodec(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  []byte
        err   error
    }{
        {"ASCII", "Hello", []byte{0x48, 0x65, ...}, nil},
        {"Extended", "€50", []byte{0x1B, 0x65, ...}, nil},
        {"Invalid", "\x00", nil, ErrInvalidChar},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := encode(tt.input)
            if err != tt.err {
                t.Errorf("error = %v, want %v", err, tt.err)
            }
            if !bytes.Equal(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**When to use:** Testing multiple input/output combinations

### Pattern 6: Zero-Value Usability

```go
// Types should work with zero values where sensible
type Transmitter struct {
    Addr        string
    EnquireLink time.Duration  // Zero value = use default
    RateLimiter RateLimiter    // Zero value (nil) = no rate limiting
}

func (t *Transmitter) init() {
    if t.EnquireLink == 0 {
        t.EnquireLink = 10 * time.Second  // Default
    }
}
```

**When to use:** Optional configuration, sensible defaults

### Pattern 7: Error Wrapping (Go 1.13+)

```go
func (c *client) Read() (pdu.Body, error) {
    p, err := c.conn.Read()
    if err != nil {
        return nil, fmt.Errorf("failed to read PDU: %w", err)
    }
    return p, nil
}

// Checking wrapped errors
if errors.Is(err, net.ErrClosed) {
    // Handle closed connection
}
```

**When to use:** Adding context to errors, preserving error types

---

## Testing Approaches

### Unit Tests

**Location:** `*_test.go` files alongside implementation

**Pattern:**
```go
func TestTransmitter_Submit(t *testing.T) {
    // Setup
    mock := &mockConn{...}
    tx := &Transmitter{conn: mock}

    // Exercise
    resp, err := tx.Submit(&ShortMessage{...})

    // Verify
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if resp.MessageID == "" {
        t.Error("expected MessageID")
    }
}
```

### Integration Tests with smpptest

```go
import "github.com/devyx-tech/go-smpp/smpp/smpptest"

func TestIntegration(t *testing.T) {
    // Create test SMSC server
    srv := smpptest.NewUnstartedServer(func(c smpp.Conn) {
        for {
            p, err := c.Read()
            if err != nil {
                return
            }
            // Respond to PDUs
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
    // ...
}
```

### Example Tests

**Location:** `example_test.go`

```go
func ExampleTransmitter_Submit() {
    tx := &smpp.Transmitter{
        Addr:   "localhost:2775",
        User:   "test",
        Passwd: "test",
    }

    conn := tx.Bind()
    <-conn

    resp, err := tx.Submit(&smpp.ShortMessage{
        Dst:  "5511999999999",
        Text: pdutext.Raw("Hello"),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.MessageID)
}
```

---

## SMPP-Specific Gotchas

### Gotcha 1: Sequence Numbers Must Be Unique Per Connection

**Problem:**
```go
// WRONG: Reusing sequence numbers
p1 := pdu.NewSubmitSM()
p1.Header().Seq = 1
conn.Write(p1)

p2 := pdu.NewSubmitSM()
p2.Header().Seq = 1  // WRONG! Duplicate sequence
conn.Write(p2)
```

**Solution:**
```go
// Correct: Use Factory for unique sequence numbers
factory := &sequenceFactory{}

p1 := pdu.NewSubmitSM()
p1.Header().Seq = factory.NewSeq()  // 1

p2 := pdu.NewSubmitSM()
p2.Header().Seq = factory.NewSeq()  // 2
```

### Gotcha 2: Data Coding Must Match Text Encoding

**Problem:**
```go
// WRONG: UCS2 text with GSM7 data_coding
sm := &smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: pdutext.UCS2("你好"),  // UCS2 encoding
}
// PDU will have data_coding=0x08 (correct)
// But if you manually set fields, mismatch can occur
```

**Solution:**
```go
// Correct: Use Codec interface (handles data_coding automatically)
sm := &smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: pdutext.UCS2("你好"),  // Codec sets data_coding=0x08
}

// Or for manual control
sm := &smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: pdutext.Raw(encodedBytes),
    // Must manually set DataCoding field if needed
}
```

### Gotcha 3: Long Messages Need UDH

**Problem:**
```go
// WRONG: Sending 200-char message without splitting
longText := strings.Repeat("A", 200)
tx.Submit(&smpp.ShortMessage{
    Text: pdutext.GSM7(longText),
})
// Will truncate or fail
```

**Solution:**
```go
// Correct: Use SubmitLongMsg for automatic splitting
longText := strings.Repeat("A", 200)
resps, err := tx.SubmitLongMsg(&smpp.ShortMessage{
    Text: pdutext.GSM7(longText),
})
// Returns multiple SubmitSMResp (one per part)
```

### Gotcha 4: EnquireLink Timeout Detection

**Problem:**
```go
// WRONG: Not monitoring EnquireLink responses
// Connection appears alive but is actually dead
```

**Solution:**
```go
// Correct: Library automatically monitors EnquireLink
tx := &smpp.Transmitter{
    EnquireLink:        10 * time.Second,  // Send every 10s
    EnquireLinkTimeout: 30 * time.Second,  // Timeout after 30s
}
// If no response in 30s, connection is closed and reconnected
```

### Gotcha 5: Bind Status Codes

**Problem:**
```go
// WRONG: Ignoring bind response status
conn := tx.Bind()
// Assuming connection is successful
tx.Submit(msg)  // Fails because bind failed
```

**Solution:**
```go
// Correct: Check bind status
conn := tx.Bind()
status := <-conn

if status.Error() != nil {
    if pduStatus, ok := status.Error().(pdu.Status); ok {
        switch pduStatus {
        case pdu.ESME_RINVPASWD:
            log.Fatal("Invalid password")
        case pdu.ESME_RALYBND:
            log.Fatal("Already bound")
        default:
            log.Fatalf("Bind failed: %v", pduStatus)
        }
    }
}
```

### Gotcha 6: Character Set Limitations

**Problem:**
```go
// WRONG: Emoji in GSM7
tx.Submit(&smpp.ShortMessage{
    Text: pdutext.GSM7("Hello 😊"),  // Emoji not in GSM7 charset
})
// Will fail or replace with ?
```

**Solution:**
```go
// Correct: Use UCS2 for emoji/international chars
tx.Submit(&smpp.ShortMessage{
    Text: pdutext.UCS2("Hello 😊"),  // UCS2 supports emoji
})

// Or auto-detect encoding
func autoEncode(text string) pdutext.Codec {
    if isGSM7(text) {
        return pdutext.GSM7(text)  // 160 chars
    }
    return pdutext.UCS2(text)  // 70 chars
}
```

### Gotcha 7: Receiver Must Respond to Deliver SM

**Problem:**
```go
// WRONG: Not sending DeliverSMResp
rx := &smpp.Receiver{
    Handler: func(p pdu.Body) {
        // Process message but don't respond
    },
}
// SMSC will timeout and may throttle or disconnect
```

**Solution:**
```go
// Correct: Library automatically sends DeliverSMResp
rx := &smpp.Receiver{
    Handler: func(p pdu.Body) {
        // Just process message
        // Library sends response automatically
    },
}
```

---

## Performance Considerations

### 1. Minimize Allocations in Hot Path

```go
// Good: Reuse buffers
var bufPool = sync.Pool{
    New: func() interface{} { return new(bytes.Buffer) },
}

func encode(data []byte) []byte {
    buf := bufPool.Get().(*bytes.Buffer)
    defer bufPool.Put(buf)
    buf.Reset()
    // Use buf
    return buf.Bytes()
}
```

### 2. Use Atomic Operations for Counters

```go
// Good: Thread-safe without mutex
type factory struct {
    seq uint32
}

func (f *factory) NewSeq() uint32 {
    return atomic.AddUint32(&f.seq, 1)
}
```

### 3. Buffered Channels for Producers/Consumers

```go
// Good: Buffered channel reduces blocking
c.Status = make(chan ConnStatus, 10)

c.notify(&connStatus{s: Connected})  // Non-blocking if buffer not full
```

### 4. Async Operations Where Possible

```go
// Good: Non-blocking notification
func (c *client) notify(ev ConnStatus) {
    select {
    case c.Status <- ev:
        // Sent
    default:
        // Channel full, drop (acceptable for status updates)
    }
}
```

---

## Common Mistakes to Avoid

### Mistake 1: Not Handling Reconnections

```go
// WRONG: Assuming connection stays up
tx := &smpp.Transmitter{...}
tx.Bind()
// Connection can drop at any time!

// Correct: Monitor status channel
go func() {
    for status := range tx.Bind() {
        if status.Status() == smpp.Disconnected {
            // Pause operations, alert monitoring
        }
    }
}()
```

### Mistake 2: Not Setting Response Timeout

```go
// WRONG: No timeout (waits forever)
tx := &smpp.Transmitter{
    // RespTimeout: 0,  // Default: 1 second
}

// Correct: Set appropriate timeout
tx := &smpp.Transmitter{
    RespTimeout: 5 * time.Second,
}
```

### Mistake 3: Blocking Handler Functions

```go
// WRONG: Slow handler blocks receiver
rx := &smpp.Receiver{
    Handler: func(p pdu.Body) {
        // Slow database write
        db.Exec("INSERT INTO messages ...")  // Blocks!
    },
}

// Correct: Process asynchronously
rx := &smpp.Receiver{
    Handler: func(p pdu.Body) {
        msg := extractMessage(p)
        go processAsync(msg)  // Non-blocking
    },
}
```

### Mistake 4: Not Closing Connections

```go
// WRONG: Connection leak
func sendSMS(msg string) error {
    tx := &smpp.Transmitter{...}
    tx.Bind()
    tx.Submit(&smpp.ShortMessage{...})
    // No Close()! Connection leaked
}

// Correct: Always close
func sendSMS(msg string) error {
    tx := &smpp.Transmitter{...}
    defer tx.Close()  // Ensures cleanup

    tx.Bind()
    return tx.Submit(&smpp.ShortMessage{...})
}
```

---

## Quick Reference: When to Use Which Client Type

```go
// Transmitter: Send SMS only
tx := &smpp.Transmitter{...}
tx.Submit(msg)

// Receiver: Receive SMS only
rx := &smpp.Receiver{
    Handler: func(p pdu.Body) { ... },
}

// Transceiver: Send AND receive on same connection
tc := &smpp.Transceiver{
    Handler: func(p pdu.Body) { ... },
}
tc.Submit(msg)
```

---

## Documentation Style

**GoDoc Format:**
```go
// Transmitter is an SMPP client for sending SMS messages.
// It maintains a persistent connection with automatic reconnection.
//
// Example usage:
//   tx := &Transmitter{Addr: "localhost:2775", User: "test", Passwd: "test"}
//   conn := tx.Bind()
//   <-conn  // Wait for connection
//   resp, err := tx.Submit(&ShortMessage{Dst: "5511999999999", Text: pdutext.Raw("Hello")})
//
// The Transmitter automatically reconnects on network failures using
// exponential backoff (max 120 seconds). Monitor connection status via
// the channel returned by Bind().
type Transmitter struct {
    // Addr is the SMSC address (e.g., "smsc.example.com:2775")
    Addr string

    // User is the SystemID for authentication
    User string
    // ...
}
```

---

## File References (For AI Context)

**Key Files by Functionality:**

- **Client Types:** `smpp/transmitter.go`, `smpp/receiver.go`, `smpp/transceiver.go`
- **Connection Management:** `smpp/client.go`, `smpp/conn.go`
- **PDU Definitions:** `smpp/pdu/*.go`
- **Character Encoding:** `smpp/pdu/pdutext/*.go`, `smpp/encoding/gsm7.go`
- **Testing Utilities:** `smpp/smpptest/*.go`
- **CLI Tools:** `cmd/sms/main.go`

**When editing code, reference these patterns from the existing codebase.**

---

**Last Updated:** 2025-10-31
