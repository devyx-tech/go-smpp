# ADR-003: Separate Client Types (Transmitter/Receiver/Transceiver)

**Status:** Accepted

**Date:** 2025-10-31 (Documented retrospectively)

**Decision Makers:** Core development team

---

## Context

SMPP 3.4 protocol defines three distinct bind operations for client connections:

### SMPP Bind Types

1. **Bind Transmitter** (Command ID: 0x00000002)
   - Purpose: Send SMS only
   - Cannot receive messages
   - Lightweight for send-only applications

2. **Bind Receiver** (Command ID: 0x00000001)
   - Purpose: Receive SMS only
   - Cannot send messages
   - Used for inbound message processing

3. **Bind Transceiver** (Command ID: 0x00000009)
   - Purpose: Send AND receive SMS
   - Bidirectional on single connection
   - Added in SMPP 3.4 for efficiency

### Problem Statement

How should the library represent these three connection types in Go?

**Options:**
1. Single unified client with mode parameter
2. Three separate struct types
3. Interface-based polymorphism
4. Functional options pattern

---

## Decision

Implement **three separate struct types**: `Transmitter`, `Receiver`, and `Transceiver`, each with type-specific methods.

### API Design

```go
// smpp/transmitter.go
type Transmitter struct {
    Addr        string
    User        string
    Passwd      string
    // Transmitter-specific fields
    WindowSize  int
    RateLimiter RateLimiter
    RespTimeout time.Duration
    // ...
}

func (t *Transmitter) Bind() <-chan ConnStatus
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error)
func (t *Transmitter) SubmitLongMsg(sm *ShortMessage) ([]*SubmitSMResp, error)
func (t *Transmitter) QuerySM(msgID, src string) (*QuerySMResp, error)
func (t *Transmitter) Close() error

// smpp/receiver.go
type Receiver struct {
    Addr    string
    User    string
    Passwd  string
    // Receiver-specific fields
    Handler              HandlerFunc
    LongMessageMerge     bool
    MergeInterval        time.Duration
    MergeCleanupInterval time.Duration
    // ...
}

func (r *Receiver) Bind() <-chan ConnStatus
func (r *Receiver) Close() error

// smpp/transceiver.go
type Transceiver struct {
    Addr    string
    User    string
    Passwd  string
    // Combined fields from both Transmitter and Receiver
    WindowSize           int
    RateLimiter          RateLimiter
    RespTimeout          time.Duration
    Handler              HandlerFunc
    LongMessageMerge     bool
    // ...
}

func (tc *Transceiver) Bind() <-chan ConnStatus
func (tc *Transceiver) Submit(sm *ShortMessage) (*SubmitSMResp, error)
func (tc *Transceiver) SubmitLongMsg(sm *ShortMessage) ([]*SubmitSMResp, error)
func (tc *Transceiver) QuerySM(msgID, src string) (*QuerySMResp, error)
func (tc *Transceiver) Close() error
```

---

## Rationale

### Why Separate Types?

**1. Type Safety**

Compile-time enforcement of protocol semantics:

```go
// Compile error: Receiver has no Submit method ✅
rx := &smpp.Receiver{...}
rx.Submit(msg)  // ERROR: rx.Submit undefined
```

With single client + mode:

```go
// Runtime error only ❌
client := &smpp.Client{Mode: ModeReceiver, ...}
client.Submit(msg)  // Panic or error at runtime
```

**2. API Clarity**

Each type exposes only relevant methods:

```go
// Clear: Transmitter can only send
tx := &smpp.Transmitter{...}
// Available: Submit, SubmitLongMsg, QuerySM

// Clear: Receiver can only receive
rx := &smpp.Receiver{...}
// Available: (none - callbacks via Handler)

// Clear: Transceiver can do both
tc := &smpp.Transceiver{...}
// Available: Submit, SubmitLongMsg, QuerySM + Handler
```

**3. Field Relevance**

Each type has only fields that make sense:

```go
// Transmitter has WindowSize (concurrent send limit)
type Transmitter struct {
    WindowSize int  // Makes sense for sending
}

// Receiver does NOT have WindowSize
type Receiver struct {
    // WindowSize omitted - not applicable
    Handler HandlerFunc  // Only for receiving
}
```

**4. Documentation Clarity**

Developers instantly understand which type to use:

- "I need to send SMS" → Use `Transmitter`
- "I need to receive SMS" → Use `Receiver`
- "I need both" → Use `Transceiver`

No need to learn mode constants or check compatibility.

**5. SMSC Provider Requirements**

Some SMSC providers require or prefer separate connections:

```go
// Provider requires separate Tx/Rx connections
tx := &smpp.Transmitter{Addr: "smsc:2775", ...}
rx := &smpp.Receiver{Addr: "smsc:2775", ...}
```

With single client, this would be awkward.

**6. Independent Configuration**

Send and receive can have different tuning:

```go
tx := &smpp.Transmitter{
    RateLimiter: rate.NewLimiter(100, 10),  // Send limit
    WindowSize:  50,
}

rx := &smpp.Receiver{
    LongMessageMerge:     true,
    MergeCleanupInterval: 5 * time.Minute,  // Receive-specific
}
```

---

## Consequences

### Positive

1. **Compile-Time Safety:**
   - Cannot call `Submit()` on `Receiver`
   - Cannot set `Handler` on `Transmitter` (not a field)
   - Type system enforces protocol semantics

2. **Clear Intent:**
   ```go
   // Intent obvious from type
   func sendSMS(tx *smpp.Transmitter, msg string) { ... }
   func receiveSMS(rx *smpp.Receiver) { ... }
   ```

3. **Better IDE Support:**
   - Autocomplete shows only relevant methods
   - Documentation split by type
   - Jump-to-definition works better

4. **Smaller Cognitive Load:**
   - No mode constants to remember
   - No "does this field apply to my mode?" questions
   - Simpler mental model

5. **Protocol Alignment:**
   - Matches SMPP spec's three bind types directly
   - Easy to explain to developers familiar with SMPP

### Negative

1. **Code Duplication:**
   - `Bind()` implementation appears in all three types
   - Common fields (`Addr`, `User`, `Passwd`) repeated
   - Some internal logic duplicated

   **Mitigation:** Internal `client` struct shared by all three types

   ```go
   // smpp/client.go (internal)
   type client struct {
       Addr   string
       User   string
       Passwd string
       // ...shared logic
   }

   // smpp/transmitter.go
   type Transmitter struct {
       Addr   string
       User   string
       Passwd string
       // Transmitter-specific fields
       WindowSize int
       // ...
       c *client  // Embedded shared client
   }
   ```

2. **Cannot Switch Modes:**
   - Must create new instance to change mode
   - Cannot reuse connection

   **Acceptable:** Rarely needed; SMPP requires rebind anyway

3. **More Types to Learn:**
   - Three types vs one
   - But each type is simpler individually

4. **Testing Complexity:**
   - Must test three types instead of one
   - More test code

   **Mitigation:** Shared test helpers

---

## Alternatives Considered

### Alternative 1: Single Client with Mode

```go
type Client struct {
    Addr string
    Mode BindMode  // ModeTransmitter, ModeReceiver, ModeTransceiver
    Handler HandlerFunc  // Only used if Mode == ModeReceiver
    WindowSize int  // Only used if Mode == ModeTransmitter
}

func (c *Client) Submit(sm *ShortMessage) error {
    if c.Mode == ModeReceiver {
        return errors.New("cannot submit in receiver mode")
    }
    // ...
}
```

**Rejected because:**
- Runtime errors instead of compile-time errors
- All fields visible regardless of relevance
- Mode constant adds cognitive load
- Documentation harder (must explain mode implications)
- IDE autocomplete less helpful (shows inapplicable methods)

### Alternative 2: Interface-Based Polymorphism

```go
type Client interface {
    Bind() <-chan ConnStatus
    Close() error
}

type Sender interface {
    Client
    Submit(*ShortMessage) error
}

type Receiver interface {
    Client
    SetHandler(HandlerFunc)
}

// Factory functions
func NewTransmitter(...) Sender
func NewReceiver(...) Receiver
func NewTransceiver(...) interface{ Sender; Receiver }
```

**Rejected because:**
- Over-engineered for this use case
- Interface indirection reduces performance (negligible but measurable)
- Harder to discover API (must read docs to find factory functions)
- Struct fields not accessible (must add getters/setters)
- Go prefers concrete types over interfaces for libraries

### Alternative 3: Functional Options Pattern

```go
type Client struct { ... }

func WithTransmitter() Option { ... }
func WithReceiver() Option { ... }
func WithHandler(h HandlerFunc) Option { ... }

client := smpp.New(
    smpp.WithTransmitter(),
    smpp.WithHandler(myHandler),
)
```

**Rejected because:**
- More complex API (must learn options)
- Still allows invalid combinations (Transmitter + Handler)
- Runtime validation needed
- Harder to document

### Alternative 4: Composition

```go
type Transmitter struct { ... }
type Receiver struct { ... }

type Transceiver struct {
    Transmitter
    Receiver
}
```

**Rejected because:**
- Two separate connections (wasteful)
- Doesn't match SMPP protocol (Transceiver is single bind)
- Field name collisions (`Addr`, `User`, etc.)

---

## Implementation Details

### Shared Internal Client

To reduce duplication, all three types use a shared internal `client` struct:

```go
// smpp/client.go
type client struct {
    Addr               string
    TLS                *tls.Config
    Status             chan ConnStatus
    BindFunc           func(c Conn) error
    EnquireLink        time.Duration
    EnquireLinkTimeout time.Duration
    RespTimeout        time.Duration
    // ...
}

func (c *client) Bind() { ... }  // Shared reconnection logic
```

Each public type wraps this:

```go
// smpp/transmitter.go
type Transmitter struct {
    Addr   string
    User   string
    Passwd string
    WindowSize int
    // ...
    c *client  // Internal
}

func (t *Transmitter) Bind() <-chan ConnStatus {
    t.c = &client{
        Addr: t.Addr,
        BindFunc: func(c Conn) error {
            // Bind as Transmitter
            return bind(c, pdu.NewBindTransmitter())
        },
        // ...
    }
    go t.c.Bind()
    return t.c.Status
}
```

**Benefits:**
- No code duplication for core logic
- Public API remains clean and type-safe
- Internal complexity hidden

### Submit Method Only on Transmitter/Transceiver

```go
// Transmitter
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error) {
    // Build SubmitSM PDU, send, await response
}

// Receiver - NO Submit method

// Transceiver - same implementation as Transmitter
func (tc *Transceiver) Submit(sm *ShortMessage) (*SubmitSMResp, error) {
    // Same logic as Transmitter
}
```

**Note:** Some code duplication between Transmitter and Transceiver `Submit()`, but acceptable trade-off for type safety.

---

## Real-World Usage Patterns

### Pattern 1: Send-Only Service (Transmitter)

```go
// Notification service sending OTPs
tx := &smpp.Transmitter{
    Addr: "smsc:2775",
    User: "notif_service",
    Passwd: "xxx",
    RateLimiter: rate.NewLimiter(100, 10),
}
tx.Bind()

http.HandleFunc("/send-otp", func(w http.ResponseWriter, r *http.Request) {
    resp, err := tx.Submit(&smpp.ShortMessage{
        Dst: r.FormValue("phone"),
        Text: pdutext.Raw(generateOTP()),
    })
    // ...
})
```

### Pattern 2: Receive-Only Service (Receiver)

```go
// Webhook service processing inbound SMS
rx := &smpp.Receiver{
    Addr: "smsc:2775",
    User: "inbound_service",
    Passwd: "xxx",
    Handler: func(p pdu.Body) {
        // Process incoming SMS
        saveToDB(p.Fields())
    },
}
rx.Bind()
select {}  // Run forever
```

### Pattern 3: Bidirectional Service (Transceiver)

```go
// Chatbot with send + receive
tc := &smpp.Transceiver{
    Addr: "smsc:2775",
    User: "chatbot",
    Passwd: "xxx",
    Handler: func(p pdu.Body) {
        // Receive message
        msg := extractMessage(p)

        // Generate reply
        reply := chatbot.Process(msg)

        // Send reply (same connection)
        tc.Submit(&smpp.ShortMessage{
            Dst: msg.From,
            Text: pdutext.Raw(reply),
        })
    },
}
tc.Bind()
select {}
```

### Pattern 4: Separate Tx/Rx (SMSC Requirement)

```go
// Some SMSCs require or recommend separate connections
tx := &smpp.Transmitter{Addr: "smsc:2775", ...}
rx := &smpp.Receiver{Addr: "smsc:2775", ...}

tx.Bind()
rx.Bind()

// Two independent connections
```

---

## Compliance

**SMPP 3.4 Specification:**
- Section 2.2.1: ESME Bind Operations
- Defines three bind types: Transmitter, Receiver, Transceiver
- Library directly models these three types

**Go Best Practices:**
- Prefer concrete types over interfaces (Effective Go)
- Use type system to enforce invariants (Go Proverbs)

---

## Related Decisions

- [ADR-001: Channel-Based Status](001-channel-based-status.md) - All three types return status channel
- [ADR-005: Interface-Based Design](005-interface-based-design.md) - Internal interfaces still used (Conn, etc.)

---

## Evolution

**Future Considerations:**

If demand exists for mode switching:

```go
// Potential future API (breaking change)
tx := &smpp.Transmitter{...}
tx.Bind()

// Upgrade to Transceiver
tc := tx.UpgradeToTransceiver(handler)
```

But currently no demand for this feature.

---

**Last Updated:** 2025-10-31
