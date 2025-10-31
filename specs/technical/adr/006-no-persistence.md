# ADR-006: No Built-in Persistence

**Status:** Accepted

**Date:** 2025-10-31 (Documented retrospectively)

**Decision Makers:** Core development team

---

## Context

SMS messaging systems typically need to persist various types of data:

### Data That Could Be Persisted

1. **Outbound Messages:**
   - Message content and destination
   - Submission status and timestamps
   - SMSC-assigned Message ID
   - Delivery receipts (DLRs)

2. **Inbound Messages:**
   - Sender, recipient, message text
   - Receive timestamp
   - Message metadata

3. **Connection State:**
   - Current bind status
   - Reconnection history
   - EnquireLink timing

4. **Message Queue:**
   - Messages waiting to be sent
   - Failed messages for retry
   - Scheduled messages

5. **Audit Logs:**
   - All SMPP PDU traffic
   - Connection events
   - Errors and status changes

### Problem Statement

Should the go-smpp library provide built-in persistence for any of these data types?

**Considerations:**
- Different applications have different storage needs (SQL, NoSQL, files, memory)
- Storage strategy affects performance, scalability, and operational complexity
- Adding persistence increases library complexity and dependencies
- Applications may already have existing storage infrastructure

---

## Decision

The library provides **zero built-in persistence** - it is completely stateless.

### What This Means

**Not Persisted by Library:**
- ❌ No database integration
- ❌ No file-based message storage
- ❌ No logging to disk
- ❌ No metrics persistence
- ❌ No connection state storage
- ❌ No message queuing

**Application Responsibility:**
- ✅ Applications choose storage backend
- ✅ Applications implement persistence logic
- ✅ Applications manage transactions and consistency
- ✅ Applications control data retention policies

---

## Rationale

### Why No Persistence?

**1. Flexibility**

Different applications need different storage:

```go
// Financial institution: PostgreSQL with audit trail
func (app *FinanceApp) SaveMessage(msg *Message) {
    app.db.Exec("INSERT INTO sms_audit (msg_id, text, timestamp) VALUES (?, ?, ?)")
}

// Startup: Redis for speed
func (app *StartupApp) SaveMessage(msg *Message) {
    app.redis.Set("msg:"+msg.ID, msg, 24*time.Hour)
}

// IoT device: SQLite on disk
func (app *IoTApp) SaveMessage(msg *Message) {
    app.sqlite.Exec("INSERT INTO messages ...")
}

// High-throughput: Kafka event stream
func (app *BigApp) SaveMessage(msg *Message) {
    app.kafka.Produce("sms-events", msg)
}
```

**If library chose one storage backend, it would force that choice on all users.**

**2. Zero Dependencies**

Current go.mod dependencies:

```go
require (
    github.com/urfave/cli v1.22.14        // CLI only
    golang.org/x/net v0.11.0              // Stdlib extension
    golang.org/x/text v0.10.0             // Character encoding
    golang.org/x/time v0.3.0              // Rate limiter interface
)
```

**With persistence, would need:**
```go
// Hypothetical - NOT included
require (
    github.com/lib/pq v1.10.9             // PostgreSQL
    github.com/go-redis/redis v9.0.0      // Redis
    github.com/mattn/go-sqlite3 v1.14.17  // SQLite (CGO!)
    // ... or pick one and force it on everyone
)
```

**Benefits of zero dependencies:**
- ✅ Faster go mod download
- ✅ Smaller binary size
- ✅ No security vulnerabilities from indirect dependencies
- ✅ No CGO requirements (SQLite)
- ✅ Easier to audit

**3. Simplicity**

Library code complexity:

```
With persistence:
- Database schema migrations
- Connection pooling
- Transaction management
- Retry logic for DB failures
- Data retention policies
- Backup and recovery
- Performance tuning
= ~5000-10000 lines of additional code

Without persistence:
= 0 lines of code
```

**4. Single Responsibility**

Unix philosophy: Do one thing well

**go-smpp responsibility:**
- ✅ Implement SMPP 3.4 protocol
- ✅ Handle network communication
- ✅ Manage connections and reconnection
- ✅ Encode/decode PDUs

**NOT go-smpp responsibility:**
- ❌ Choose storage backend
- ❌ Manage database schema
- ❌ Implement data retention
- ❌ Provide audit trails

**5. Performance Control**

Storage performance is critical:

```go
// Fast path: No storage blocking Submit()
resp, err := tx.Submit(msg)
go saveToDB(resp)  // Async, non-blocking

// vs.

// Hypothetical with built-in persistence
resp, err := tx.Submit(msg)
// Library blocks on database write
// Application cannot optimize
```

Applications can choose:
- Async storage (fire and forget)
- Batched writes (bulk inserts)
- Buffered channels (backpressure)
- Write-behind cache (Redis → PostgreSQL)

Library cannot make these choices for all users.

**6. Operational Complexity**

With built-in persistence:

```yaml
# Docker Compose - hypothetical
services:
  smpp-client:
    image: myapp
    environment:
      - DB_HOST=postgres
      - DB_USER=smpp
      - DB_PASSWORD=xxx
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7
```

**Without built-in persistence:**

```yaml
services:
  smpp-client:
    image: myapp
    # No database required if app doesn't need persistence
```

Simpler deployments, fewer moving parts.

---

## Consequences

### Positive

1. **Maximum Flexibility:**
   - Applications choose storage that fits their needs
   - Can change storage without changing library
   - Multiple storage backends in same app (e.g., Redis + PostgreSQL)

2. **Zero Dependencies:**
   - Smaller binary
   - Faster compilation
   - No indirect dependency vulnerabilities
   - No CGO requirements

3. **Simpler Library:**
   - Focused on SMPP protocol
   - Easier to maintain
   - Fewer bugs (no database edge cases)
   - Clearer separation of concerns

4. **Better Performance:**
   - No forced synchronous I/O
   - Applications control storage batching
   - Can skip storage entirely (e.g., transient notifications)

5. **Easier Testing:**
   - No database mocking needed for library tests
   - Applications test persistence separately
   - Cleaner test boundaries

### Negative

1. **More Application Code:**
   - Every application must implement persistence
   - Risk of inconsistent implementations
   - Each team reinvents solutions

   **Mitigation:** Provide examples in documentation

2. **No Consistency Guarantees:**
   - Library cannot ensure messages are persisted before sending
   - Applications must implement transactional guarantees

   **Mitigation:** Document patterns for reliable persistence

3. **Learning Curve:**
   - Developers must design their own persistence layer
   - Must understand SMPP message lifecycle

   **Mitigation:** Comprehensive integration examples

### Neutral

1. **Stateless on Crash:**
   - Inflight messages lost if process dies
   - No automatic recovery

   **Acceptable:** SMS protocol is inherently unreliable (network failures, SMSC crashes); applications requiring reliability must implement persistence themselves

---

## Alternatives Considered

### Alternative 1: Built-in SQL Persistence

```go
type Transmitter struct {
    DB *sql.DB  // Required
    // ...
}

func (t *Transmitter) Submit(sm *ShortMessage) error {
    // Library saves to DB before sending
    tx := t.DB.Begin()
    tx.Exec("INSERT INTO messages ...")
    err := t.sendToSMSC(sm)
    if err != nil {
        tx.Rollback()
        return err
    }
    tx.Commit()
    return nil
}
```

**Rejected because:**
- ❌ Forces SQL on all users (no NoSQL, no memory-only, no custom storage)
- ❌ Adds `database/sql` complexity
- ❌ Library must define schema (breaks flexibility)
- ❌ Blocking I/O on every Submit() (performance impact)
- ❌ Transaction management complexity
- ❌ Migrations and versioning needed

### Alternative 2: Pluggable Storage Interface

```go
type MessageStorage interface {
    Save(*Message) error
    Load(id string) (*Message, error)
    Delete(id string) error
}

type Transmitter struct {
    Storage MessageStorage  // Optional
}

// Implementations
type SQLStorage struct { db *sql.DB }
type RedisStorage struct { client *redis.Client }
type NoOpStorage struct {}  // No persistence
```

**Rejected because:**
- ❌ Still adds complexity (interface design, testing)
- ❌ Must maintain "official" implementations (SQL, Redis, etc.)
- ❌ Version compatibility (storage interface changes break implementations)
- ❌ Limited flexibility (interface may not fit all use cases)
- ❌ Applications can implement this pattern themselves if needed

### Alternative 3: Event Hooks

```go
type Transmitter struct {
    OnBeforeSubmit func(*ShortMessage)
    OnAfterSubmit  func(*ShortMessage, *SubmitSMResp, error)
    OnReceive      func(*DeliverSM)
}

// Usage
tx := &smpp.Transmitter{
    OnAfterSubmit: func(sm *ShortMessage, resp *SubmitSMResp, err error) {
        saveToDatabase(sm, resp, err)
    },
}
```

**Rejected because:**
- ❌ Adds API surface area
- ❌ Callback timing is tricky (before? after? async?)
- ❌ Error handling complexity (what if callback fails?)
- ❌ Applications can already do this with wrapper functions:

   ```go
   // Application can wrap Submit() if needed
   func (app *App) Submit(sm *smpp.ShortMessage) error {
       app.saveToDBBefore(sm)
       resp, err := app.tx.Submit(sm)
       app.saveToDBAfter(sm, resp, err)
       return err
   }
   ```

### Alternative 4: Write-Ahead Log (WAL)

```go
// Library maintains append-only log of all operations
type Transmitter struct {
    WALPath string  // /var/lib/smpp/wal.log
}

func (t *Transmitter) Submit(sm *ShortMessage) error {
    t.wal.Append(sm)  // Write to disk
    // Send to SMSC
}
```

**Rejected because:**
- ❌ Forces file I/O on all users
- ❌ Log rotation and cleanup needed
- ❌ Disk space management
- ❌ What about read-only filesystems? (containers)
- ❌ Cross-platform file permissions issues
- ❌ Applications needing WAL can implement themselves

---

## Recommended Patterns (Application Layer)

### Pattern 1: Async Persistence

```go
// Non-blocking: save to DB asynchronously
resp, err := tx.Submit(msg)
if err != nil {
    return err
}

go func() {
    db.Exec("INSERT INTO sent_messages (msg_id, ...) VALUES (?, ...)", resp.MessageID)
}()

return nil  // Don't wait for DB
```

**Use When:** Throughput > consistency

### Pattern 2: Write-Behind Cache

```go
// Fast: write to Redis, sync to Postgres in background
resp, err := tx.Submit(msg)
redis.Set("msg:"+resp.MessageID, msg)

// Background worker syncs Redis → Postgres
go func() {
    for {
        msgs := redis.GetPending()
        db.BulkInsert(msgs)
        redis.DeleteSynced(msgs)
    }
}()
```

**Use When:** High throughput, eventual consistency OK

### Pattern 3: Transactional Outbox

```go
// Reliable: database transaction ensures consistency
tx := db.Begin()
tx.Exec("INSERT INTO outbox (msg, status) VALUES (?, 'pending')", msg)
tx.Commit()

// Background worker sends pending messages
go func() {
    pending := db.Query("SELECT * FROM outbox WHERE status='pending'")
    for _, msg := range pending {
        resp, err := smppTx.Submit(msg)
        if err == nil {
            db.Exec("UPDATE outbox SET status='sent', msg_id=? WHERE id=?", resp.MessageID, msg.ID)
        }
    }
}()
```

**Use When:** Reliability > throughput

### Pattern 4: Event Sourcing

```go
// Append-only log of all events
type Event struct {
    Type      string  // "submitted", "received", "delivered"
    MessageID string
    Payload   []byte
    Timestamp time.Time
}

resp, err := tx.Submit(msg)
eventStore.Append(Event{
    Type:      "submitted",
    MessageID: resp.MessageID,
    Payload:   json.Marshal(msg),
    Timestamp: time.Now(),
})

// Rebuild state by replaying events
```

**Use When:** Audit trail required, event-driven architecture

### Pattern 5: No Persistence (Acceptable!)

```go
// Transient notifications: no storage needed
resp, err := tx.Submit(&smpp.ShortMessage{
    Dst:  user.Phone,
    Text: pdutext.Raw("Your OTP: " + otp),
})

// Log MessageID for debugging, but don't persist message
log.Printf("Sent OTP to %s, MessageID=%s", user.Phone, resp.MessageID)

return resp.MessageID
```

**Use When:** Messages are ephemeral, reliability not critical

---

## Integration Examples

See [docs/integrations.md](../../docs/integrations.md) for complete examples:
- PostgreSQL integration
- Redis caching
- RabbitMQ queuing
- Event sourcing patterns

---

## Compliance

**12-Factor App:**
- Factor 6: "Processes are stateless and share-nothing"
- go-smpp is stateless by design

**Microservices Patterns:**
- Database per Service: Applications own their data schema
- Saga Pattern: Applications coordinate distributed transactions

---

## Related Decisions

- [ADR-004: In-Memory Message Merging](004-in-memory-message-merging.md) - Merge state not persisted
- [ADR-002: Automatic Reconnection](002-automatic-reconnection.md) - Inflight messages lost on disconnect

---

## Future Considerations

**Will NOT Add:**
- Built-in database support
- Required storage interface
- Persistence hooks in core API

**Might Add (Separate Package):**
- Optional `smpp/storage` package with reference implementations
- Separate Go module to avoid adding dependencies to core
- Community-maintained adapters (Redis, PostgreSQL, etc.)

Example:
```go
// Separate module: github.com/devyx-tech/go-smpp-storage
import "github.com/devyx-tech/go-smpp-storage/postgres"

store := postgres.New(db)
resp, err := tx.Submit(msg)
store.Save(resp)  // Optional, not in core library
```

But even this is low priority - applications can easily implement themselves.

---

**Last Updated:** 2025-10-31
