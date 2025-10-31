# ADR-002: Automatic Reconnection with Exponential Backoff

**Status:** Accepted

**Date:** 2025-10-31 (Documented retrospectively)

**Decision Makers:** Core development team

---

## Context

SMPP connections over TCP are inherently unreliable in production environments:

### Real-World Connection Failures

1. **Network Issues:**
   - Mobile network instability
   - Firewall timeouts (idle connections dropped)
   - NAT gateway session expiration
   - Packet loss in lossy networks

2. **SMSC Server Issues:**
   - Planned maintenance/restarts
   - Server crashes or failures
   - Load balancer health check failures
   - Configuration changes

3. **Protocol Issues:**
   - EnquireLink timeout (no response from SMSC)
   - Bind rejection (credentials expired, account suspended)
   - SMSC-initiated unbind

### Problem Statement

When a connection fails, the library must decide:
- Should it automatically reconnect or require manual intervention?
- How frequently should it attempt reconnection?
- How to avoid overwhelming a failed SMSC with reconnection attempts?
- How to balance responsiveness with resource conservation?

---

## Decision

Implement **automatic reconnection with exponential backoff** capped at 120 seconds.

### Reconnection Strategy

```go
// Pseudocode
delay := 1.0  // Initial delay: 1 second
maxDelay := 120.0  // Maximum delay: 2 minutes

for !closed() {
    err := connect()
    if err == nil {
        delay = 1.0  // Reset on successful connection
        // Connected - run until failure
        continue
    }

    // Connection failed
    sleep(delay)
    delay = min(delay * e, maxDelay)  // e ≈ 2.718 (Euler's number)
}
```

### Actual Implementation

Location: `smpp/client.go:132-189`

```go
func (c *client) Bind() {
    delay := 1.0
    const maxdelay = 120.0

    for !c.closed() {
        conn, err := Dial(c.Addr, c.TLS)
        if err != nil {
            c.notify(&connStatus{s: ConnectionFailed, err: err})
            goto retry
        }

        c.conn.Set(conn)
        if err = c.BindFunc(c.conn); err != nil {
            c.notify(&connStatus{s: BindFailed, err: err})
            goto retry
        }

        c.notify(&connStatus{s: Connected})
        delay = 1  // Reset backoff on success

        // Run until connection fails
        for {
            p, err := c.conn.Read()
            if err != nil {
                c.notify(&connStatus{s: Disconnected, err: err})
                break
            }
            // Process PDU...
        }

    retry:
        c.conn.Close()
        delay = math.Min(delay*math.E, maxdelay)
        c.trysleep(time.Duration(delay) * time.Second)
    }
    close(c.Status)
}
```

---

## Rationale

### Why Automatic Reconnection?

**Production Reality:**
- SMS traffic is often time-sensitive (OTPs, alerts)
- Manual intervention is too slow (humans sleep, may not notice immediately)
- Connection failures are often transient (network blips, server restarts)
- High availability requirements demand automatic recovery

**User Experience:**
- Developers expect resilient libraries that handle failures gracefully
- Applications should focus on business logic, not connection management
- Reduces operational burden (fewer alerts, manual restarts)

### Why Exponential Backoff?

**Exponential Growth Curve:**
```
Attempt   Delay (seconds)   Cumulative Time
1         1                 1s
2         2.7               3.7s
3         7.4               11.1s
4         20.0              31.1s
5         54.6              85.7s
6         120 (capped)      205.7s
7+        120 (capped)      ...
```

**Benefits:**
1. **Fast Initial Recovery:**
   - First reconnect after just 1 second
   - Transient failures recovered quickly
   - Minimal impact on message delivery

2. **Backpressure on Failed SMSC:**
   - Exponential growth reduces load on struggling SMSC
   - Caps at 120s prevent indefinite growth
   - Respects SMSC recovery time

3. **Resource Conservation:**
   - Fewer connection attempts = less CPU/network usage
   - Prevents connection storm (many clients reconnecting simultaneously)

4. **Industry Standard:**
   - Used by AWS, Google Cloud, and most resilient systems
   - Well-understood by operations teams

### Why 120 Second Cap?

**Chosen Value:** 2 minutes maximum backoff

**Reasoning:**
- Short enough for reasonable recovery time (< 5 minutes to detect and reconnect after extended outage)
- Long enough to avoid overwhelming failed SMSC
- Matches typical SMSC restart time (1-3 minutes)

**Alternatives Considered:**
- **30s cap:** Too aggressive; could overload recovering SMSC
- **300s cap (5min):** Too slow; SMS delivery SLAs often < 5 minutes
- **No cap:** Risk of hours-long delays (unacceptable for SMS)

---

## Consequences

### Positive

1. **High Availability:**
   - Automatic recovery from transient failures
   - No manual intervention required
   - Applications remain operational during SMSC outages

2. **SMSC-Friendly:**
   - Exponential backoff prevents connection storms
   - Respects server recovery time
   - Reduces load on struggling infrastructure

3. **Developer Experience:**
   - Zero configuration required (sensible defaults)
   - Optional monitoring via status channel
   - Reduces operational complexity

4. **Production Proven:**
   - Industry-standard pattern
   - Predictable behavior under failure
   - Easy to reason about for operations teams

### Negative

1. **Unbounded Retries:**
   - Library never gives up (keeps trying forever)
   - Applications must call `Close()` to stop reconnection
   - Could waste resources if SMSC is permanently offline

   **Mitigation:** Status channel allows applications to detect repeated failures and take action

2. **Bind Failures Also Retry:**
   - Even authentication failures trigger reconnection
   - Could lock account with repeated failed attempts

   **Mitigation:** Exponential backoff limits attempt frequency; applications should monitor BindFailed status

3. **No Jitter:**
   - All clients reconnect at same intervals
   - Risk of thundering herd if many clients disconnect simultaneously

   **Mitigation:** Could add jitter in future if needed; not observed as problem in practice

4. **Inflight Messages Lost:**
   - Pending requests in inflight map are lost on disconnect
   - Applications receive timeout errors

   **Mitigation:** Applications should implement retry logic for critical messages

### Neutral

1. **Resource Usage:**
   - Background goroutine runs continuously (acceptable)
   - Connection attempts use minimal CPU/network
   - Memory usage is constant (no state accumulation)

---

## Alternatives Considered

### Alternative 1: No Automatic Reconnection

```go
func (t *Transmitter) Bind() error {
    // Connect once, return error on failure
    // Application must manually reconnect
}
```

**Rejected because:**
- High operational burden (manual intervention required)
- Poor developer experience
- Not production-ready for high-availability systems
- Most applications would implement their own reconnection logic anyway

### Alternative 2: Fixed Interval Retry

```go
// Retry every 5 seconds indefinitely
for !closed() {
    connect()
    time.Sleep(5 * time.Second)
}
```

**Rejected because:**
- Wastes resources (too many connection attempts)
- Can overwhelm failed SMSC
- Doesn't adapt to failure duration
- Not respectful of infrastructure

### Alternative 3: Limited Retry Count

```go
const maxRetries = 10

for attempts := 0; attempts < maxRetries; attempts++ {
    err := connect()
    if err == nil {
        break
    }
    time.Sleep(backoff(attempts))
}
```

**Rejected because:**
- Arbitrary retry limit
- Long outages exceed retry count
- Requires application to restart connection after limit
- SMS services need indefinite availability

### Alternative 4: Configurable Backoff Parameters

```go
type Transmitter struct {
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    BackoffMultiplier float64
}
```

**Rejected because:**
- Adds configuration complexity
- Sensible defaults work for 99% of use cases
- Applications rarely need custom backoff
- Could be added later if demand exists

---

## Implementation Details

### Backoff Calculation

```go
delay := 1.0  // seconds
const maxdelay = 120.0

// After each failure
delay = math.Min(delay * math.E, maxdelay)
```

**Why Euler's number (e ≈ 2.718)?**
- Smoother growth than doubling (2×)
- Reaches max delay in ~6 attempts
- Used in Go standard library (net/http retries)

### Shutdown Handling

```go
func (c *client) Close() error {
    c.once.Do(func() {
        close(c.stop)  // Signal reconnection loop to exit
        // Send unbind, wait briefly, close connection
    })
    return nil
}

func (c *client) trysleep(d time.Duration) {
    select {
    case <-time.After(d):
        // Sleep completed
    case <-c.stop:
        // Shutdown requested, exit immediately
    }
}
```

**Key Points:**
- `Close()` interrupts sleep immediately
- No long delays during graceful shutdown
- Status channel closed to signal end of stream

### EnquireLink Timeout

**Separate from Reconnection:**

EnquireLink keepalive detects stale connections:

```go
// If no EnquireLinkResp received in 3× EnquireLink interval,
// close connection and trigger reconnection
if time.Since(c.eliTime) >= c.EnquireLinkTimeout {
    c.conn.Close()  // Triggers reconnection loop
    return
}
```

This ensures connections don't hang indefinitely waiting for data.

---

## Compliance

**SMPP 3.4 Spec:**
- Does not mandate reconnection behavior
- Library design choice for production robustness

**Industry Best Practices:**
- Exponential backoff: RFC 6585 (HTTP), AWS SDK, Google Cloud
- Automatic reconnection: Standard for network protocols (HTTP/2, gRPC, MQTT)

---

## Related Decisions

- [ADR-001: Channel-Based Status Notifications](001-channel-based-status.md) - Status channel reports reconnection events
- [ADR-006: No Built-in Persistence](006-no-persistence.md) - Inflight messages lost on disconnect

---

## Observability

Applications should monitor reconnection frequency:

```go
reconnectCount := 0

go func() {
    for status := range tx.Bind() {
        if status.Status() == smpp.Connected {
            reconnectCount++
            if reconnectCount > 10 {
                // Alert: Frequent reconnections indicate problem
                alertOps("SMPP reconnecting frequently")
            }
        }
    }
}()
```

**Metrics to Track:**
- Reconnection frequency (reconnects/hour)
- Time spent disconnected (downtime)
- Bind failure rate (authentication issues)
- Connection lifespan (time between disconnects)

---

## Future Enhancements

**Possible Improvements (Not Implemented):**

1. **Jitter:** Add random ±10% variation to backoff to prevent thundering herd
2. **Configurable Backoff:** Allow custom backoff parameters if demand exists
3. **Circuit Breaker:** Stop reconnecting after N consecutive bind failures (requires breaking change)
4. **Backoff Reset:** Reset delay to 1s after X seconds of successful connection (not just on reconnect)

---

**Last Updated:** 2025-10-31
