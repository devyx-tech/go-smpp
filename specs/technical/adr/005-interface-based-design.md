# ADR-005: Interface-Based Design for Extensibility

**Status:** Accepted

**Date:** 2025-10-31 (Documented retrospectively)

**Decision Makers:** Core development team

---

## Context

The go-smpp library must balance several competing concerns:

1. **Testability:** Enable easy mocking and testing without real SMSC connections
2. **Extensibility:** Allow applications to customize behavior (logging, metrics, custom transports)
3. **Performance:** Minimize indirection overhead
4. **Simplicity:** Avoid over-abstraction that hurts usability
5. **Go Idioms:** Follow "accept interfaces, return structs" principle

### Problem Statement

Which parts of the library should be abstracted behind interfaces, and which should be concrete types?

---

## Decision

Use **selective interface abstraction** for key extension points while keeping high-level API as concrete structs.

### Interface Boundaries

**Interfaces Defined:**

1. **`Conn` - Connection Abstraction**
   ```go
   // smpp/client.go:27-32
   type Conn interface {
       Read() (pdu.Body, error)
       Write(pdu.Body) error
       Close() error
   }
   ```

2. **`ConnMiddleware` - Connection Interceptor**
   ```go
   // smpp/client.go:74
   type ConnMiddleware func(Conn) Conn
   ```

3. **`RateLimiter` - Throughput Control**
   ```go
   // smpp/client.go:84-87
   type RateLimiter interface {
       Wait(ctx context.Context) error
   }
   ```

4. **`pdu.Body` - PDU Abstraction**
   ```go
   // smpp/pdu/pdu.go:15-20
   type Body interface {
       Header() *Header
       Fields() pdufield.Map
       TLVFields() pdutlv.Map
       SerializeTo(io.Writer) error
   }
   ```

5. **`pdu.Factory` - Sequence Number Generation**
   ```go
   // smpp/pdu/pdu.go:21-23
   type Factory interface {
       NewSeq() uint32
   }
   ```

6. **`pdutext.Codec` - Text Encoding Strategy**
   ```go
   // smpp/pdu/pdutext/codec.go:15-20
   type Codec interface {
       Type() DataCoding
       Encode() []byte
       Decode() []byte
   }
   ```

**Concrete Types (No Interfaces):**
- `Transmitter`, `Receiver`, `Transceiver` (high-level API)
- `ShortMessage` (data structure)
- All response types (`SubmitSMResp`, `QuerySMResp`, etc.)

---

## Rationale

### Why `Conn` Interface?

**Enables:**

1. **Testing without SMSC:**
   ```go
   type mockConn struct {
       responses []pdu.Body
   }

   func (m *mockConn) Read() (pdu.Body, error) {
       return m.responses[0], nil
   }

   func (m *mockConn) Write(p pdu.Body) error {
       return nil
   }
   ```

2. **Custom Transports:**
   - Unix domain sockets
   - SSH tunnels
   - WebSocket proxies
   - Mock servers for integration tests

3. **Middleware Composition:**
   ```go
   type loggingConn struct {
       next Conn
   }

   func (c *loggingConn) Write(p pdu.Body) error {
       log.Printf("Sending: %v", p.Header().ID)
       return c.next.Write(p)
   }
   ```

**Cost:**
- Interface call overhead: ~1-2ns (negligible for network I/O)
- One indirection in hot path (acceptable trade-off)

### Why `RateLimiter` Interface?

**Enables:**

1. **Integration with `golang.org/x/time/rate`:**
   ```go
   limiter := rate.NewLimiter(10, 5)
   tx := &smpp.Transmitter{
       RateLimiter: limiter,  // Implements interface
   }
   ```

2. **Custom Rate Limiting:**
   ```go
   type adaptiveRateLimiter struct {
       // Dynamic rate adjustment based on SMSC response
   }

   func (a *adaptiveRateLimiter) Wait(ctx context.Context) error {
       // Custom logic
   }
   ```

3. **Multi-Tier Limits:**
   ```go
   type compositeRateLimiter struct {
       perSecond  RateLimiter
       perMinute  RateLimiter
       perHour    RateLimiter
   }
   ```

**Cost:**
- Zero if not used (`RateLimiter` field can be nil)
- Single interface call in `Submit()` hot path (acceptable)

### Why `Codec` Interface?

**Enables:**

1. **Multiple Encoding Strategies:**
   - GSM7, Latin1, UCS2, ISO-8859-5, Raw
   - Applications can add custom encodings

2. **Runtime Codec Selection:**
   ```go
   func chooseCodec(text string) pdutext.Codec {
       if isGSM7(text) {
           return pdutext.GSM7(text)
       }
       return pdutext.UCS2(text)
   }

   sm := &smpp.ShortMessage{
       Text: chooseCodec(message),
   }
   ```

3. **Testability:**
   ```go
   type spyCodec struct {
       encodeCalls int
   }

   func TestEncoding(t *testing.T) {
       spy := &spyCodec{}
       sm.Text = spy
       tx.Submit(sm)
       assert.Equal(t, 1, spy.encodeCalls)
   }
   ```

**Cost:**
- Interface call during PDU construction (not in hot path - once per message)

### Why `pdu.Body` Interface?

**Enables:**

1. **Uniform PDU Handling:**
   ```go
   func process(p pdu.Body) {
       // Works with any PDU type
       seq := p.Header().Seq
       fields := p.Fields()
   }
   ```

2. **Mock PDUs for Testing:**
   ```go
   type mockPDU struct {
       header *pdu.Header
       fields pdufield.Map
   }
   ```

3. **Custom PDU Types:**
   - Applications can implement proprietary extensions
   - Future SMPP versions (5.0) support

**Cost:**
- All PDU types implement interface via embedding (zero runtime cost)

### Why No Interface for `Transmitter`?

**Principle:** Accept interfaces, return concrete structs

```go
// Good: Accept Conn interface
func bindTransmitter(c Conn, user, passwd string) error {
    // Can test with mock Conn
}

// Good: Return concrete Transmitter
func NewTransmitter(addr string) *Transmitter {
    // Caller gets full API access
}

// Bad: Return Transmitter interface
func NewTransmitter(addr string) Sender {
    // Forces interface indirection for common operations
}
```

**Reasons:**
- `Transmitter` is the primary API - should be concrete for performance and discoverability
- Applications rarely need to mock entire Transmitter (use `Conn` mock instead)
- Struct fields (configuration) need direct access
- Interface would hide struct fields (require getters/setters)

---

## Consequences

### Positive

1. **Testability:**
   ```go
   func TestTransmitter(t *testing.T) {
       mock := &mockConn{}
       // Inject mock via middleware or direct construction
   }
   ```

2. **Extensibility:**
   - Middleware for logging/metrics
   - Custom rate limiters
   - Custom text encodings
   - Alternative transports

3. **Composability:**
   - Interfaces stack naturally (middleware chain)
   - Compatible with standard library patterns

4. **Type Safety:**
   - Concrete types catch errors at compile time
   - IDEs provide better autocomplete

5. **Performance:**
   - Interface calls only in non-critical paths
   - Zero-cost abstractions where not used

### Negative

1. **Learning Curve:**
   - Developers must understand when to use interface vs concrete type
   - Documentation must explain extension points

2. **Testing Complexity:**
   - Must understand which interfaces to mock
   - Mock implementations require boilerplate

   **Mitigation:** Provide test helpers in `smpptest` package

3. **Potential for Over-Abstraction:**
   - Easy to add unnecessary interfaces
   - Must resist premature abstraction

   **Mitigation:** Only add interfaces when extension point is needed

---

## Interface Design Principles

### 1. Small Interfaces

Following Go proverbs: "The bigger the interface, the weaker the abstraction"

```go
// Good: 3 methods
type Conn interface {
    Read() (pdu.Body, error)
    Write(pdu.Body) error
    Close() error
}

// Good: 1 method
type RateLimiter interface {
    Wait(ctx context.Context) error
}

// Bad: Too many methods (hypothetical)
type Client interface {
    Bind() error
    Submit(*ShortMessage) error
    QuerySM(string, string) error
    Close() error
    SetHandler(func(pdu.Body))
    GetStatus() ConnStatus
    // ... 10 more methods
}
```

### 2. Accept Interfaces, Return Structs

```go
// Good: Function accepts interface
func logConnection(c Conn) Conn {
    return &loggingConn{next: c}
}

// Good: Function returns struct
func NewTransmitter(...) *Transmitter {
    return &Transmitter{...}
}
```

### 3. Interface at Usage Site

Define interfaces where they're used, not where they're implemented:

```go
// smpp/client.go - defines Conn interface
type Conn interface { ... }

func (c *client) Bind(conn Conn) { ... }  // Uses interface

// smpp/conn.go - implements interface
type connImpl struct { ... }

func (c *connImpl) Read() (pdu.Body, error) { ... }  // No explicit "implements"
```

### 4. Zero-Value Interfaces

```go
// RateLimiter can be nil (zero value)
type Transmitter struct {
    RateLimiter RateLimiter  // Optional
}

func (t *Transmitter) Submit(sm *ShortMessage) error {
    if t.RateLimiter != nil {
        t.RateLimiter.Wait(ctx)  // Only call if set
    }
    // ...
}
```

---

## Real-World Extension Examples

### Example 1: Prometheus Metrics Middleware

```go
type metricsConn struct {
    next   smpp.Conn
    sent   prometheus.Counter
    received prometheus.Counter
}

func (m *metricsConn) Write(p pdu.Body) error {
    m.sent.Inc()
    return m.next.Write(p)
}

func (m *metricsConn) Read() (pdu.Body, error) {
    p, err := m.next.Read()
    if err == nil {
        m.received.Inc()
    }
    return p, err
}

func NewMetricsMiddleware(reg prometheus.Registerer) smpp.ConnMiddleware {
    sent := prometheus.NewCounter(...)
    received := prometheus.NewCounter(...)
    reg.MustRegister(sent, received)

    return func(next smpp.Conn) smpp.Conn {
        return &metricsConn{next: next, sent: sent, received: received}
    }
}

// Usage
tx := &smpp.Transmitter{
    Middleware: NewMetricsMiddleware(prometheus.DefaultRegisterer),
}
```

### Example 2: Retry Rate Limiter

```go
type retryRateLimiter struct {
    base    *rate.Limiter
    retries int
}

func (r *retryRateLimiter) Wait(ctx context.Context) error {
    for i := 0; i < r.retries; i++ {
        err := r.base.Wait(ctx)
        if err == nil {
            return nil
        }
        time.Sleep(time.Second * time.Duration(i))
    }
    return errors.New("rate limit exceeded after retries")
}

// Usage
tx := &smpp.Transmitter{
    RateLimiter: &retryRateLimiter{
        base:    rate.NewLimiter(10, 5),
        retries: 3,
    },
}
```

### Example 3: Custom Text Encoding

```go
type cp1252Codec struct {
    text string
}

func (c *cp1252Codec) Type() pdutext.DataCoding {
    return 0x03  // Latin1 data coding
}

func (c *cp1252Codec) Encode() []byte {
    // Custom CP1252 encoding logic
    return encodeCP1252(c.text)
}

func (c *cp1252Codec) Decode() []byte {
    return []byte(c.text)
}

// Usage
sm := &smpp.ShortMessage{
    Dst:  "5511999999999",
    Text: &cp1252Codec{text: "Olá com CP1252"},
}
```

---

## Testing Strategy

### Mock Interfaces for Unit Tests

```go
// smpptest/mock_conn.go
type MockConn struct {
    ReadFunc  func() (pdu.Body, error)
    WriteFunc func(pdu.Body) error
    CloseFunc func() error
}

func (m *MockConn) Read() (pdu.Body, error) {
    return m.ReadFunc()
}

func (m *MockConn) Write(p pdu.Body) error {
    return m.WriteFunc(p)
}

func (m *MockConn) Close() error {
    return m.CloseFunc()
}

// Test usage
func TestSubmit(t *testing.T) {
    conn := &MockConn{
        WriteFunc: func(p pdu.Body) error {
            assert.Equal(t, pdu.SubmitSMID, p.Header().ID)
            return nil
        },
        ReadFunc: func() (pdu.Body, error) {
            return pdu.NewSubmitSMResp(), nil
        },
    }

    // Test with mock connection
}
```

---

## Compliance

**Effective Go:**
- "Accept interfaces, return concrete types"
- "The bigger the interface, the weaker the abstraction"
- "A little copying is better than a little dependency"

**Go Proverbs:**
- "interface{} says nothing"
- "Design for testability"

---

## Related Decisions

- [ADR-003: Separate Client Types](003-separate-client-types.md) - Why `Transmitter` is concrete, not interface
- [ADR-001: Channel-Based Status](001-channel-based-status.md) - `ConnStatus` is interface for flexibility

---

## Evolution

**Future Interface Candidates:**

If demand exists, consider interfaces for:
- `MessageStorage` - for persistent message buffering
- `ConnectionPool` - for managing multiple SMSC connections
- `RoutingStrategy` - for multi-SMSC routing logic

But only add when real extension need is demonstrated.

---

**Last Updated:** 2025-10-31
