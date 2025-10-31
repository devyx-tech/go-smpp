# Troubleshooting Guide - go-smpp

**Purpose:** Common issues, SMSC provider compatibility problems, debugging tips, and resolution strategies.

---

## Connection Issues

### Issue: "connection refused" or "dial tcp: connection refused"

**Symptoms:** Cannot establish TCP connection to SMSC

**Causes:**
- SMSC is down or unreachable
- Wrong address or port
- Firewall blocking connection
- Network issue

**Solutions:**
```bash
# Test connectivity
telnet smsc.example.com 2775
nc -zv smsc.example.com 2775

# Check DNS resolution
nslookup smsc.example.com
dig smsc.example.com

# Test with curl (if HTTP available)
curl -v telnet://smsc.example.com:2775
```

**Code Fix:**
```go
// Add timeout to detect issues faster
tx := &smpp.Transmitter{
    Addr:        "smsc.example.com:2775",
    EnquireLink: 5 * time.Second,  // Faster detection
}

// Monitor status
conn := tx.Bind()
status := <-conn
if status.Error() != nil {
    log.Printf("Connection error: %v", status.Error())
}
```

---

### Issue: "bind failed" or "invalid password"

**Symptoms:** Connection succeeds but bind fails with status code

**Causes:**
- Invalid credentials (SystemID or Password)
- Account suspended or expired
- IP not whitelisted
- Wrong system_type or interface_version

**SMPP Status Codes:**
- `0x0000000E`: ESME_RINVPASWD (Invalid password)
- `0x0000000F`: ESME_RINVSYSID (Invalid system ID)
- `0x00000051`: ESME_RALYBND (Already bound)

**Solutions:**
```go
// Check bind response
conn := tx.Bind()
status := <-conn

if status.Error() != nil {
    if pduStatus, ok := status.Error().(pdu.Status); ok {
        switch pduStatus {
        case pdu.ESME_RINVPASWD:
            log.Fatal("Invalid password - check credentials")
        case pdu.ESME_RINVSYSID:
            log.Fatal("Invalid SystemID")
        case pdu.ESME_RALYBND:
            log.Fatal("Already bound - close existing connection")
        }
    }
}
```

**Contact SMSC provider to:**
- Verify credentials
- Check account status
- Whitelist your IP
- Confirm bind type (Transmitter/Receiver/Transceiver)

---

### Issue: Connection drops frequently

**Symptoms:** Frequent Disconnected status notifications

**Causes:**
- Network instability
- Firewall killing idle connections
- SMSC-side timeout
- EnquireLink timeout too aggressive

**Solutions:**
```go
// Increase EnquireLink timeout
tx := &smpp.Transmitter{
    EnquireLink:        15 * time.Second,  // Send every 15s
    EnquireLink Timeout: 60 * time.Second,  // Timeout after 60s
}

// Monitor reconnections
reconnectCount := 0
go func() {
    for status := range tx.Bind() {
        if status.Status() == smpp.Connected {
            reconnectCount++
            if reconnectCount > 10 {
                log.Printf("WARNING: Reconnected %d times", reconnectCount)
            }
        }
    }
}()
```

**Check:**
- Firewall rules (may kill idle TCP connections)
- NAT gateway timeout settings
- SMSC provider connection limits

---

## Message Submission Issues

### Issue: "invalid destination address" (ESME_RINVDSTADR)

**Symptoms:** Submit fails with 0x0000000B status

**Causes:**
- Invalid phone number format
- Wrong TON/NPI combination
- Number not reachable by SMSC
- Country code missing/incorrect

**Solutions:**
```go
// Try different TON/NPI combinations
sm := &smpp.ShortMessage{
    Dst:         "5511999999999",
    DestAddrTON: 1,  // International
    DestAddrNPI: 1,  // ISDN
    Text:        pdutext.Raw("Test"),
}

// Or unknown (let SMSC decide)
sm := &smpp.ShortMessage{
    Dst:         "5511999999999",
    DestAddrTON: 0,  // Unknown
    DestAddrNPI: 0,  // Unknown
    Text:        pdutext.Raw("Test"),
}

// Validate number format before submitting
func validateE164(phone string) bool {
    // E.164: +[country][number]
    matched, _ := regexp.MatchString(`^\+?[1-9]\d{1,14}$`, phone)
    return matched
}
```

---

### Issue: "throttled" (ESME_RTHROTTLED)

**Symptoms:** Submit fails with 0x00000058 status

**Causes:**
- Exceeded SMSC rate limit
- Sending too fast
- SMSC under load

**Solutions:**
```go
import "golang.org/x/time/rate"

// Add rate limiting
limiter := rate.NewLimiter(10, 5)  // 10 msg/s, burst 5

tx := &smpp.Transmitter{
    RateLimiter: limiter,
}

// Retry with backoff on throttle
resp, err := tx.Submit(sm)
if status, ok := err.(pdu.Status); ok {
    if status == pdu.ESME_RTHROTTLED {
        time.Sleep(1 * time.Second)
        resp, err = tx.Submit(sm)  // Retry
    }
}
```

---

### Issue: "timeout waiting for response"

**Symptoms:** ErrTimeout error after submission

**Causes:**
- SMSC slow to respond
- Network latency
- SMSC overloaded
- Response lost in network

**Solutions:**
```go
// Increase response timeout
tx := &smpp.Transmitter{
    RespTimeout: 10 * time.Second,  // Default: 1s
}

// Implement retry logic
func submitWithRetry(tx *smpp.Transmitter, sm *smpp.ShortMessage, retries int) error {
    for i := 0; i < retries; i++ {
        resp, err := tx.Submit(sm)
        if err == smpp.ErrTimeout {
            log.Printf("Timeout, retry %d/%d", i+1, retries)
            time.Sleep(time.Duration(i) * time.Second)
            continue
        }
        return err
    }
    return smpp.ErrTimeout
}
```

---

## Character Encoding Issues

### Issue: Message arrives with "?" or garbled characters

**Symptoms:** SMS displays incorrectly on phone

**Causes:**
- Wrong data_coding
- Character not in selected encoding
- SMSC transcoding issues

**Solutions:**
```go
// Use UCS2 for international/emoji
tx.Submit(&smpp.ShortMessage{
    Text: pdutext.UCS2("你好 😊"),  // Unicode
})

// Check if text fits encoding
func isGSM7Compatible(text string) bool {
    // Check against GSM7 character table
    for _, r := range text {
        if !isInGSM7Table(r) {
            return false
        }
    }
    return true
}

// Auto-select encoding
func autoEncode(text string) pdutext.Codec {
    if isGSM7Compatible(text) {
        return pdutext.GSM7(text)  // 160 chars
    }
    return pdutext.UCS2(text)  // 70 chars
}
```

---

### Issue: Message truncated at 160/70 characters

**Symptoms:** Long messages cut off

**Causes:**
- Not using `SubmitLongMsg()`
- SMSC doesn't support long messages

**Solutions:**
```go
// Use SubmitLongMsg for automatic splitting
longText := strings.Repeat("A", 200)
resps, err := tx.SubmitLongMsg(&smpp.ShortMessage{
    Text: pdutext.GSM7(longText),
})
// Returns multiple SubmitSMResp

// Or manually split
func splitMessage(text string, maxLen int) []string {
    var parts []string
    for len(text) > 0 {
        if len(text) <= maxLen {
            parts = append(parts, text)
            break
        }
        parts = append(parts, text[:maxLen])
        text = text[maxLen:]
    }
    return parts
}
```

---

## SMSC Provider Compatibility

### Data Coding Scheme Variations

**Problem:** Different SMSCs interpret `data_coding=0` differently

**Solutions:**
```go
// Test with specific SMSC
encodings := []pdutext.Codec{
    pdutext.GSM7("Test"),
    pdutext.Raw("Test"),  // Try as plain ASCII
    pdutext.Latin1("Test"),
}

for _, enc := range encodings {
    resp, err := tx.Submit(&smpp.ShortMessage{
        Dst:  testNumber,
        Text: enc,
    })
    log.Printf("Encoding %d: %v", enc.Type(), err)
}
```

**Document per-SMSC requirements** in application configuration

---

### TON/NPI Requirements

**Problem:** Some SMSCs require specific TON/NPI

**Example Configurations:**
```go
// International E.164
sm.SourceAddrTON = 1  // International
sm.SourceAddrNPI = 1  // ISDN

// Alphanumeric sender
sm.SourceAddrTON = 5  // Alphanumeric
sm.SourceAddrNPI = 0  // Unknown

// Short code
sm.SourceAddrTON = 3  // Network specific
sm.SourceAddrNPI = 0
```

---

## Debugging Tips

### Enable PDU Logging

```go
func loggingMiddleware(next smpp.Conn) smpp.Conn {
    return &loggingConn{next: next}
}

type loggingConn struct {
    next smpp.Conn
}

func (c *loggingConn) Write(p pdu.Body) error {
    h := p.Header()
    log.Printf("→ TX: ID=0x%08X Seq=%d Status=0x%08X", h.ID, h.Seq, h.Status)

    // Log fields
    for k, v := range p.Fields() {
        log.Printf("  %s: %v", k, v)
    }

    return c.next.Write(p)
}

func (c *loggingConn) Read() (pdu.Body, error) {
    p, err := c.next.Read()
    if err == nil {
        h := p.Header()
        log.Printf("← RX: ID=0x%08X Seq=%d Status=0x%08X", h.ID, h.Seq, h.Status)
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

### Packet Capture with Wireshark

```bash
# Capture SMPP traffic
sudo tcpdump -i any -s 0 -w smpp.pcap 'port 2775'

# Analyze with Wireshark
wireshark smpp.pcap

# Filter in Wireshark: smpp
```

### Connection State Monitoring

```go
// Track all status changes
go func() {
    for status := range tx.Bind() {
        log.Printf("Status: %s, Error: %v, Time: %s",
            status.Status(),
            status.Error(),
            time.Now().Format(time.RFC3339))
    }
}()
```

---

## Performance Issues

### High Latency

**Causes:**
- Network distance to SMSC
- SMSC overloaded
- WindowSize too small

**Solutions:**
```go
// Increase concurrent requests
tx := &smpp.Transmitter{
    WindowSize: 100,  // Default: 0 (unlimited)
}

// Use multiple connections
for i := 0; i < 5; i++ {
    tx := &smpp.Transmitter{...}
    go func() {
        tx.Bind()
        // Each connection sends messages
    }()
}
```

### Memory Growth

**Causes:**
- Incomplete long messages not cleaned up
- Inflight map growing

**Solutions:**
```go
// Reduce cleanup interval
rx := &smpp.Receiver{
    MergeCleanupInterval: 2 * time.Minute,  // Default: 5min
}

// Monitor memory
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
// Access http://localhost:6060/debug/pprof/heap
```

---

## Common Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| `connection refused` | SMSC not reachable | Check network, firewall, SMSC address |
| `invalid password` | Wrong credentials | Verify SystemID and Password |
| `timeout waiting for response` | SMSC slow/not responding | Increase RespTimeout, check SMSC load |
| `not connected` | Connection not established | Wait for Bind() status channel |
| `invalid destination address` | Bad phone number | Validate number format, check TON/NPI |
| `throttled` | Rate limit exceeded | Add rate limiting, slow down |
| `message too long` | Message exceeds limit | Use SubmitLongMsg() or split manually |

---

## Getting Support

**Check existing issues:** https://github.com/devyx-tech/go-smpp/issues

**When reporting issues, include:**
- Go version (`go version`)
- Library version/commit
- SMSC provider (if not sensitive)
- Minimal reproducible example
- Error messages and logs (sanitize sensitive data)
- PDU logs (use logging middleware)

---

**Last Updated:** 2025-10-31
