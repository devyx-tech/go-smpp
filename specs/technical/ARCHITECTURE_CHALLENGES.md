# Architecture Challenges - go-smpp

**Purpose:** Document current limitations, known issues, scalability considerations, SMSC provider quirks, and future improvement areas.

---

## Current Limitations

### 1. No Persistent State

**Limitation:** Library is completely stateless

**Impact:**
- Inflight messages lost on crash
- Incomplete long messages discarded on restart
- No automatic recovery of pending operations

**Workaround:**
- Applications must implement persistence layer
- Use transactional outbox pattern for reliability
- See [ADR-006: No Built-in Persistence](adr/006-no-persistence.md)

**Future Consideration:**
- Optional storage interface (separate module)
- Reference implementations for common databases

---

### 2. Unimplemented PDU Types

**Not Implemented:**
- `outbind` - Server-initiated bind (rarely used)
- `alert_notification` - Availability alerts (rarely used)
- `data_sm` / `data_sm_resp` - Alternative data transfer (rarely used)
- `cancel_sm` / `cancel_sm_resp` - Cancel scheduled message (rarely used)
- `replace_sm` / `replace_sm_resp` - Replace scheduled message (rarely used)

**Impact:**
- Cannot cancel or replace scheduled messages
- Cannot receive server-initiated binds
- Cannot use DataSM alternative to SubmitSM

**Rationale:**
- These PDUs are rarely used in practice (< 1% of deployments)
- Adding them increases complexity with minimal benefit
- Most SMSCs don't fully support these operations

**Future:** Implement on demand if use case emerges

---

### 3. No Connection Pooling

**Limitation:** Each Transmitter/Receiver/Transceiver manages single connection

**Impact:**
- Applications must manually create multiple instances for connection pooling
- No built-in load balancing across connections
- No automatic failover between connections

**Workaround:**
```go
// Manual connection pool
type SMPPPool struct {
    transmitters []*smpp.Transmitter
    next         uint32
}

func (p *SMPPPool) Submit(sm *smpp.ShortMessage) error {
    idx := atomic.AddUint32(&p.next, 1) % uint32(len(p.transmitters))
    return p.transmitters[idx].Submit(sm)
}
```

**Future Consideration:**
- Separate connection pool package
- Automatic load balancing
- Health-based routing

---

### 4. No Multi-SMSC Routing

**Limitation:** Library connects to single SMSC endpoint

**Impact:**
- No automatic failover to backup SMSC
- No routing based on destination (country, carrier)
- No cost-based routing

**Workaround:**
- Application-layer routing logic
- Multiple Transmitter instances
- Custom routing table

**Example:**
```go
type Router struct {
    primary   *smpp.Transmitter
    backup    *smpp.Transmitter
    routes    map[string]*smpp.Transmitter  // country code -> SMSC
}

func (r *Router) Submit(sm *smpp.ShortMessage) error {
    // Try primary
    err := r.primary.Submit(sm)
    if err != nil {
        // Fallback to backup
        return r.backup.Submit(sm)
    }
    return err
}
```

---

### 5. No Built-in Retry Logic

**Limitation:** Failed submissions not automatically retried

**Impact:**
- Application must implement retry logic
- Risk of message loss on transient errors
- Complex error handling

**Workaround:**
```go
func submitWithRetry(tx *smpp.Transmitter, sm *smpp.ShortMessage) error {
    backoff := time.Second
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := tx.Submit(sm)
        if err == nil {
            return nil
        }

        // Retry on timeout or throttle
        if err == smpp.ErrTimeout || err == pdu.ESME_RTHROTTLED {
            time.Sleep(backoff)
            backoff *= 2
            continue
        }

        return err  // Permanent error
    }
    return errors.New("max retries exceeded")
}
```

**Future Consideration:**
- Optional retry middleware
- Configurable retry policy

---

## Scalability Considerations

### Throughput Limits

**Single Connection Limits:**
- Typical SMSC window size: 10-100 concurrent requests
- Observed throughput: 50-1000 messages/second per connection
- Limited by network latency and SMSC processing time

**Scaling Strategies:**
1. **Horizontal Scaling:**
   ```go
   // Multiple connections
   for i := 0; i < 10; i++ {
       tx := &smpp.Transmitter{...}
       go worker(tx, msgQueue)
   }
   ```

2. **Multiple Instances:**
   - Deploy multiple application instances
   - Each with own SMSC connections
   - Use load balancer for distribution

3. **Async Processing:**
   - Message queue (RabbitMQ, Kafka)
   - Worker pool consuming from queue
   - Database for persistence

**Bottlenecks:**
- SMSC capacity
- Network bandwidth
- Rate limits imposed by SMSC

---

### Memory Usage

**Per Connection:**
- Base client: ~10-50 KB
- Inflight map: ~1 KB per pending message
- Merge buffer: ~1 KB per incomplete long message
- Typical total: ~100 KB - 1 MB per connection

**Memory Growth:**
- Inflight map grows with WindowSize
- Merge buffer grows with incomplete messages
- Cleanup timers limit maximum growth

**Monitoring:**
```go
// Track inflight count
func (t *Transmitter) InflightCount() int {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return len(t.inflight)
}

// Alert if too high
if tx.InflightCount() > 1000 {
    log.Warn("High inflight count, possible SMSC slowness")
}
```

---

## SMSC Provider Quirks

### 1. Data Coding Variations

**Issue:** Different interpretation of `data_coding` field

**Providers:**
- **Provider A:** `data_coding=0` → GSM7
- **Provider B:** `data_coding=0` → ASCII
- **Provider C:** `data_coding=0` → ISO-8859-1
- **Provider D:** Requires `data_coding=241` (custom)

**Solution:** Test and document per-provider requirements

```go
// Provider-specific config
type ProviderConfig struct {
    DataCodingGSM7  uint8  // Provider's value for GSM7
    DataCodingUCS2  uint8  // Provider's value for UCS2
    RequiresTON     bool   // Strict TON/NPI validation
    SupportsLongMsg bool   // Long message support
}

providers := map[string]ProviderConfig{
    "providerA": {DataCodingGSM7: 0x00, ...},
    "providerB": {DataCodingGSM7: 0x00, ...},
}
```

---

### 2. TON/NPI Strictness

**Issue:** Some SMSCs require exact TON/NPI, others are lenient

**Strict Providers:**
- Reject messages with wrong TON/NPI
- Require international format for TON=1

**Lenient Providers:**
- Accept TON=0, NPI=0 (unknown)
- Auto-detect format

**Solution:**
```go
// Try unknown first, fall back to specific
sm := &smpp.ShortMessage{
    Dst:         dest,
    DestAddrTON: 0,  // Unknown
    DestAddrNPI: 0,
}

_, err := tx.Submit(sm)
if status, ok := err.(pdu.Status); ok && status == pdu.ESME_RINVDSTADR {
    // Retry with international
    sm.DestAddrTON = 1
    sm.DestAddrNPI = 1
    _, err = tx.Submit(sm)
}
```

---

### 3. Validity Period Format

**Issue:** Not all SMSCs support both absolute and relative time

**Formats:**
- **Absolute:** `YYMMDDhhmmsstnnp` (specific date/time)
- **Relative:** `YYMMDDhhmmss000R` (duration from now)

**Solution:** Check SMSC documentation, use supported format

---

### 4. Long Message Support

**Issue:** UDH concatenation support varies

**Implementations:**
- **Full Support:** Receives all parts, delivers to phone in order
- **No Support:** Each part delivered as separate SMS
- **Partial Support:** Sometimes works, sometimes doesn't
- **Requires Configuration:** Must enable on SMSC side

**Detection:**
```go
// Test long message support
longText := strings.Repeat("A", 200)
resps, err := tx.SubmitLongMsg(&smpp.ShortMessage{
    Dst:  testNumber,
    Text: pdutext.GSM7(longText),
})

// Check if all parts submitted successfully
if err == nil && len(resps) > 1 {
    log.Printf("Long messages supported, %d parts", len(resps))
} else {
    log.Printf("Long messages may not be supported")
}
```

---

### 5. Window Size Enforcement

**Issue:** SMSC window size limits vary

**Common Values:**
- 10 (conservative)
- 50 (moderate)
- 100 (aggressive)
- Unlimited (rare)

**Impact:** Exceeding window causes throttling or rejection

**Solution:**
```go
// Start conservative, increase if no errors
tx := &smpp.Transmitter{
    WindowSize: 10,  // Start here
}

// Monitor and adjust
if throttleCount == 0 && successRate > 0.99 {
    // Increase window
    tx.WindowSize = 50
}
```

---

## Known Issues

### 1. Race Condition in Rapid Close/Reopen

**Scenario:** Calling `Close()` immediately followed by new `Bind()` on same struct

**Impact:** Potential goroutine leak or panic

**Workaround:** Create new instance instead of reusing

```go
// Don't do this
tx.Close()
tx.Bind()  // Undefined behavior

// Do this instead
tx.Close()
tx = &smpp.Transmitter{...}  // New instance
tx.Bind()
```

**Status:** Low priority (rare use case)

---

### 2. No Graceful Degradation on Partial Submit

**Scenario:** SubmitMulti to 100 destinations, 1 fails

**Impact:** Entire operation fails, unclear which destinations succeeded

**Workaround:** Submit individually if reliability critical

```go
// Individual submissions for reliability
for _, dest := range destinations {
    resp, err := tx.Submit(&smpp.ShortMessage{
        Dst:  dest,
        Text: text,
    })
    // Track success/failure per destination
}
```

**Future:** Return partial success results

---

### 3. Cleanup Timer Not Configurable Per-Message

**Scenario:** Some long messages may need longer cleanup timeout

**Impact:** Messages taking > 5 minutes to arrive are discarded

**Workaround:** Increase global cleanup interval

```go
rx := &smpp.Receiver{
    MergeCleanupInterval: 15 * time.Minute,  // Longer timeout
}
```

**Future:** Per-message timeout based on first part timestamp

---

## Future Improvement Areas

### High Priority

1. **Connection Pool Interface:**
   - Abstract connection pooling
   - Load balancing strategies
   - Health-based routing

2. **Retry Middleware:**
   - Configurable retry policy
   - Exponential backoff
   - Circuit breaker pattern

3. **Observability Hooks:**
   - Metrics interface (Prometheus-compatible)
   - Tracing integration (OpenTelemetry)
   - Structured logging hooks

### Medium Priority

4. **DataSM Support:**
   - Implement DataSM PDU
   - Alternative to SubmitSM
   - Larger payload support

5. **Async Batch Operations:**
   - Batch submission API
   - Bulk commit for efficiency

6. **Enhanced Error Types:**
   - Rich error context
   - Error categories
   - Retry decision helpers

### Low Priority

7. **SMPP 5.0 Support:**
   - Newer protocol version
   - Additional features
   - Backward compatibility

8. **Advanced Routing:**
   - Multi-SMSC routing
   - Cost-based selection
   - Geo-based routing

9. **Admin Operations:**
   - Connection statistics API
   - Runtime reconfiguration
   - Graceful shutdown API

---

## Migration Paths

### From Stateless to Stateful

**If persistence added in future:**

```go
// Optional storage backend
import "github.com/devyx-tech/go-smpp/storage/postgres"

tx := &smpp.Transmitter{
    Storage: postgres.New(db),  // Optional
}

// Backward compatible: nil storage = current behavior
```

### Adding Missing PDU Types

**Non-breaking addition:**

```go
// New methods on existing types
func (t *Transmitter) CancelSM(msgID string) error {
    // New functionality
}

// Existing code continues to work
```

---

## Recommendations for Production

### 1. Implement Application-Level Persistence

- Store messages before submission
- Update status after SMSC confirmation
- Retry failed messages

### 2. Monitor Connection Health

- Track reconnection frequency
- Alert on repeated failures
- Log all status changes

### 3. Rate Limit Conservatively

- Start with known SMSC limits
- Gradually increase while monitoring
- Back off on throttle errors

### 4. Test with Specific SMSC

- Verify encoding support
- Test long messages
- Check TON/NPI requirements
- Measure throughput limits

### 5. Plan for Failure

- Implement retry logic
- Use message queues
- Have backup SMSC configured
- Monitor delivery rates

---

## Contributing Improvements

**To address these challenges:**

1. Open issue describing problem and proposed solution
2. Discuss approach with maintainers
3. Implement with tests and documentation
4. Submit pull request

**See [CONTRIBUTING.md](CONTRIBUTING.md) for process**

---

**Last Updated:** 2025-10-31
