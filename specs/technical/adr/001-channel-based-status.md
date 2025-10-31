# ADR-001: Channel-Based Status Notifications

**Status:** Accepted

**Date:** 2025-10-31 (Documented retrospectively)

**Decision Makers:** Core development team

---

## Context

The SMPP client needs to notify the application about connection status changes (connected, disconnected, connection failed, bind failed). The library must support:

1. Asynchronous status updates without blocking operations
2. Multiple concurrent SMS submissions while monitoring connection health
3. Graceful handling of reconnection events
4. Non-blocking application code that can choose to monitor or ignore status

### Problem Statement

How should the library communicate connection status changes to the consuming application in a way that:
- Doesn't block message submission operations
- Allows applications to react to status changes asynchronously
- Supports Go's concurrency model idiomatically
- Enables optional monitoring (apps can ignore if they don't care)

---

## Decision

Use **Go channels** (`<-chan ConnStatus`) returned from the `Bind()` method to communicate status changes asynchronously.

### API Design

```go
type ConnStatus interface {
    Status() ConnStatusID
    Error() error
}

type ConnStatusID uint8

const (
    Connected ConnStatusID = iota + 1
    Disconnected
    ConnectionFailed
    BindFailed
)

// Bind returns a channel that receives status updates
func (t *Transmitter) Bind() <-chan ConnStatus
```

### Usage Pattern

```go
tx := &smpp.Transmitter{Addr: "smsc.example.com:2775", ...}

// Get status channel
conn := tx.Bind()

// Option 1: Block until first connection
status := <-conn
if status.Error() != nil {
    log.Fatal("Connection failed:", status.Error())
}

// Option 2: Monitor in background goroutine
go func() {
    for status := range conn {
        switch status.Status() {
        case smpp.Connected:
            log.Println("Connected to SMSC")
        case smpp.Disconnected:
            log.Println("Disconnected, reconnecting...")
        }
    }
}()

// Main application continues with SMS submission
```

---

## Rationale

### Why Channels Over Callbacks?

**Channels (Chosen):**
- ✅ Idiomatic Go concurrency pattern
- ✅ Non-blocking by default
- ✅ Application controls consumption (select, goroutines)
- ✅ No callback registration API needed
- ✅ Easy to compose with other channels (select statements)
- ✅ Type-safe without reflection

**Callbacks (Not Chosen):**
- ❌ Requires callback registration API
- ❌ Potential for blocking library internals if callback is slow
- ❌ Harder to compose with Go's concurrency primitives
- ❌ Not idiomatic Go

**Polling (Not Chosen):**
- ❌ Wasteful (busy-wait)
- ❌ Latency in detecting status changes
- ❌ Requires mutex-protected state access

### Why Read-Only Channel?

The `Bind()` method returns `<-chan ConnStatus` (receive-only) rather than `chan ConnStatus` to:
- Prevent applications from accidentally sending to the channel
- Make ownership clear (library writes, application reads)
- Follow Go convention for producer-consumer patterns

### Why Separate from Submit()?

Status notifications are **asynchronous events** unrelated to individual message submissions. Separating concerns allows:
- Applications to submit messages without checking status every time
- Background monitoring of connection health
- Automatic reconnection without affecting Submit() API

---

## Consequences

### Positive

1. **Non-Blocking Operations:**
   - Applications can submit SMS without waiting for status updates
   - Status monitoring happens asynchronously in separate goroutine

2. **Flexible Consumption:**
   ```go
   // Ignore status (rely on automatic reconnection)
   tx.Bind()

   // Block until connected
   <-tx.Bind()

   // Monitor continuously
   go func() {
       for status := range tx.Bind() {
           // Handle status
       }
   }()
   ```

3. **Composability:**
   ```go
   // Can use select with multiple channels
   select {
   case status := <-conn:
       handleStatus(status)
   case msg := <-msgQueue:
       tx.Submit(msg)
   case <-time.After(timeout):
       // Handle timeout
   }
   ```

4. **Clear Semantics:**
   - Receive-only channel makes ownership obvious
   - Channel closure indicates shutdown

### Negative

1. **Buffered Channel Management:**
   - Library must use buffered channel (capacity 10) to avoid blocking status updates
   - If application doesn't consume, older status updates are dropped (acceptable trade-off)

2. **Channel Closure Semantics:**
   - Channel closes when `Close()` is called
   - Applications must handle channel closure in loops

3. **Learning Curve:**
   - Developers unfamiliar with Go channels may initially find this confusing
   - Documentation and examples mitigate this

### Neutral

1. **Memory Overhead:**
   - One channel per client connection (~few KB)
   - Negligible in practice

---

## Alternatives Considered

### Alternative 1: Callback Function

```go
type StatusCallback func(status ConnStatus)

type Transmitter struct {
    OnStatusChange StatusCallback
}
```

**Rejected because:**
- Not idiomatic Go
- Callback must not block or library stalls
- Harder to test
- Requires mutex protection if callback modifies shared state

### Alternative 2: Sync Method with Blocking

```go
func (t *Transmitter) Bind() error {
    // Blocks until connected or fails
}
```

**Rejected because:**
- No way to monitor reconnections after initial bind
- Application blocks during reconnection attempts
- Forces synchronous programming model

### Alternative 3: Polling Method

```go
func (t *Transmitter) Status() ConnStatus {
    // Application polls periodically
}
```

**Rejected because:**
- Wasteful busy-waiting
- Latency in detecting status changes
- Requires mutex-protected access to status field

### Alternative 4: Observable Pattern (RxGo)

```go
func (t *Transmitter) Bind() rx.Observable {
    // Returns RxGo observable
}
```

**Rejected because:**
- Introduces heavy external dependency
- Overkill for simple status notifications
- Not idiomatic Go

---

## Implementation Details

### Channel Buffering

```go
// client.go
type client struct {
    Status chan ConnStatus // Buffered channel (capacity 10)
    // ...
}

func (c *client) init() {
    c.Status = make(chan ConnStatus, 10)
    // ...
}

func (c *client) notify(ev ConnStatus) {
    select {
    case c.Status <- ev:
        // Sent successfully
    default:
        // Channel full, drop oldest (acceptable)
    }
}
```

### Automatic Reconnection Loop

```go
func (c *client) Bind() {
    for !c.closed() {
        conn, err := Dial(c.Addr, c.TLS)
        if err != nil {
            c.notify(&connStatus{s: ConnectionFailed, err: err})
            // Exponential backoff
            c.trysleep(delay)
            continue
        }

        c.notify(&connStatus{s: Connected})

        // Read loop until connection fails
        for {
            p, err := conn.Read()
            if err != nil {
                c.notify(&connStatus{s: Disconnected, err: err})
                break
            }
            // Process PDU
        }
    }
    close(c.Status) // Signal shutdown
}
```

---

## Compliance

This decision aligns with:
- **Go Concurrency Patterns:** https://go.dev/blog/pipelines
- **Effective Go:** https://go.dev/doc/effective_go#channels
- **SMPP 3.4 Spec:** No specific requirement; library design choice

---

## Related Decisions

- [ADR-002: Automatic Reconnection](002-automatic-reconnection.md) - Uses status channel to report reconnection events
- [ADR-005: Interface-Based Design](005-interface-based-design.md) - ConnStatus is an interface for testability

---

## Notes

### Performance Characteristics

- **Channel Operations:** O(1) time complexity
- **Memory:** ~few KB per channel (negligible)
- **Goroutine Overhead:** 1 background goroutine per connection (acceptable)

### Testing

Channel-based design makes testing easy:

```go
func TestStatusNotifications(t *testing.T) {
    tx := &Transmitter{...}
    conn := tx.Bind()

    // Wait for first status
    status := <-conn

    if status.Status() != Connected {
        t.Error("Expected Connected status")
    }
}
```

---

**Last Updated:** 2025-10-31
