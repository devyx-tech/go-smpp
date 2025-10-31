# Project Charter - go-smpp

## Project Vision

Provide a **production-ready, idiomatic Go implementation** of the SMPP 3.4 protocol that enables developers to build reliable SMS communication microservices with minimal boilerplate and maximum resilience.

### Mission Statement

Build the most reliable and developer-friendly SMPP library in the Go ecosystem by:
- Implementing full SMPP 3.4 protocol compliance
- Providing automatic reconnection and fault tolerance
- Enabling high-throughput concurrent operations
- Maintaining zero external dependencies for core functionality
- Offering comprehensive text encoding support

---

## Project Objectives

### Primary Objectives

1. **Protocol Compliance**
   - 100% compliance with SMPP 3.4 specification
   - Support for all critical PDU types (Bind, Submit, Deliver, Query, EnquireLink)
   - Correct handling of TLV (Tag-Length-Value) optional fields

2. **Production Reliability**
   - Automatic reconnection with exponential backoff
   - Thread-safe concurrent operations
   - Connection health monitoring via keepalive (EnquireLink)
   - Graceful error handling and status reporting

3. **Developer Experience**
   - Idiomatic Go API with clean interfaces
   - Channel-based async communication patterns
   - Comprehensive examples and documentation
   - Easy integration into microservices

4. **Performance**
   - Support for high-throughput message submission
   - Asynchronous request/response correlation
   - Optional rate limiting interface
   - Minimal memory footprint

---

## Success Criteria

### Functional Success Criteria

✅ **Must Have:**
- Successfully bind to any SMPP 3.4 compliant SMSC
- Send SMS messages (SubmitSM) with delivery confirmation
- Receive SMS messages (DeliverSM) via handler callbacks
- Handle long messages (>160 chars) with automatic UDH concatenation
- Support multiple character encodings (GSM7, Latin1, UCS2, ISO-8859-5)
- Automatic reconnection on connection failure
- Query message delivery status (QuerySM)

✅ **Should Have:**
- TLS/SSL secure connections
- Rate limiting support
- Multiple destination submission (SubmitMulti)
- Middleware for logging/metrics
- CLI tools for testing

⚠️ **Nice to Have:**
- DataSM support (currently not implemented)
- CancelSM/ReplaceSM operations (rarely used)
- Alert notifications (rarely used)

### Non-Functional Success Criteria

✅ **Achieved:**
- Zero external dependencies for core SMPP functionality
- Cross-platform support (Linux, macOS, Windows)
- Thread-safe concurrent operations
- Battle-tested in production environments
- Comprehensive test coverage

---

## Scope Boundaries

### In Scope

**Protocol Implementation:**
- SMPP 3.4 protocol PDU encoding/decoding
- Transmitter, Receiver, Transceiver client types
- Connection management and keepalive
- Automatic reconnection logic
- Character encoding transformations

**Developer Features:**
- Public API for SMS submission and reception
- Status notification channels
- Error handling and SMPP status codes
- Rate limiter interface
- Connection middleware hooks
- CLI tools for testing

**Testing:**
- Unit tests for protocol components
- Integration tests with mock SMSC (smpptest package)
- Example code and documentation

### Out of Scope

**Not Implemented by Design:**
- **Message Persistence:** Library is stateless; persistence is the responsibility of consuming applications
- **Message Routing:** No intelligent routing; connects to single SMSC endpoint
- **Multi-SMSC Load Balancing:** Application layer responsibility
- **Database Integration:** No built-in storage
- **HTTP/REST API:** Library provides Go API only; wrapping in HTTP is application responsibility
- **Authentication Beyond SMPP:** Only SystemID/Password supported (per SMPP spec)
- **Message Queuing:** No built-in queue; use external queue systems
- **SMPP Server Implementation:** Library is client-only (except test server for testing)

**Protocol Limitations:**
- DataSM PDU (not implemented - rarely used)
- CancelSM/ReplaceSM PDU (not implemented - rarely used)
- AlertNotification PDU (not implemented - rarely used)
- Outbind (server-initiated bind - not implemented)

---

## Key Stakeholders

### Primary Stakeholders

**Development Teams:**
- Go microservice developers building SMS notification services
- Backend engineers integrating SMPP into existing systems
- DevOps teams deploying SMS gateway services

**Business Use Cases:**
- Transactional SMS (OTP, notifications, alerts)
- Marketing campaigns (bulk SMS)
- Two-way SMS communication (customer service)
- IoT device communication

### Secondary Stakeholders

**SMSC Providers:**
- Telecom operators providing SMPP access
- SMS gateway aggregators
- Cloud SMS service providers

**End Users (Indirect):**
- Mobile phone users receiving SMS
- Businesses sending SMS communications

---

## Technical Constraints

### Protocol Constraints

**SMPP 3.4 Specification:**
- PDU header must be exactly 16 bytes
- Maximum PDU size typically 4096 bytes (SMSC-dependent)
- Sequence numbers must be unique per connection
- Async window size limits (typically 10-100 concurrent requests)

**Character Encoding Limits:**
- GSM7: 160 chars (153 with UDH)
- Latin1/UCS2: 70 chars (67 with UDH)
- Binary: 140 bytes (134 with UDH)

**Protocol Timing:**
- EnquireLink typically every 10-30 seconds
- Response timeout typically 1-10 seconds
- Bind timeout typically 5-30 seconds

### Platform Constraints

**Go Version:**
- Minimum Go 1.18 (for generics support)
- Standard library networking (net, io, context)
- No CGO dependencies

**Network Requirements:**
- TCP connectivity to SMSC (usually port 2775)
- Optional TLS support (usually port 2776)
- Bidirectional traffic for transceiver mode

**Resource Constraints:**
- Memory: ~10-50MB per connection (depending on window size)
- CPU: Minimal (network I/O bound)
- Goroutines: ~3-5 per active connection

### Operational Constraints

**Deployment:**
- Must support containerized deployment (Docker/Kubernetes)
- Should handle network partitions gracefully
- Must survive SMSC restarts without data loss (for in-flight messages)

**Observability:**
- Library does not include built-in telemetry
- Applications must implement logging/metrics via middleware
- Status channels provide connection state visibility

**Scalability:**
- Horizontal scaling via multiple instances
- Each instance manages independent connection(s)
- No shared state between instances

---

## Architecture Constraints

### Design Principles

**Idiomatic Go:**
- Follow Go best practices and conventions
- Use interfaces for extensibility
- Leverage channels for async communication
- Minimal use of reflection

**Separation of Concerns:**
- Protocol layer independent of transport
- Encoding layer separate from PDU handling
- Client logic separate from connection management

**Fail-Fast vs Resilient:**
- Validation errors fail fast (e.g., invalid destination)
- Network errors trigger automatic reconnection
- Protocol errors reported via status codes

### Technology Decisions

**Core Dependencies:**
- `golang.org/x/text` - Character encoding transformations
- `golang.org/x/time` - Rate limiting interface
- `golang.org/x/net` - Network utilities
- `github.com/urfave/cli` - CLI tool framework (cmd/ only)

**Standard Library Usage:**
- `net` - TCP/TLS connections
- `encoding/binary` - PDU binary serialization
- `context` - Timeout and cancellation
- `sync` - Concurrency primitives
- `time` - Timers and timeouts

---

## Risk Assessment

### Technical Risks

**High Risk:**
- **SMSC Provider Variations:** Different providers interpret SMPP spec differently (especially data_coding)
  - **Mitigation:** Extensive testing with multiple providers, flexible configuration

- **Network Instability:** Mobile networks can be unreliable
  - **Mitigation:** Automatic reconnection, exponential backoff, keepalive monitoring

**Medium Risk:**
- **Character Encoding Edge Cases:** Unusual character combinations may not encode correctly
  - **Mitigation:** Comprehensive encoding tests, clear documentation of supported characters

- **Race Conditions:** Concurrent access to shared state
  - **Mitigation:** Extensive use of mutexes, atomic operations, channel synchronization

**Low Risk:**
- **Memory Leaks:** Long-running connections accumulating state
  - **Mitigation:** Regular cleanup of inflight maps, merge buffers; periodic testing

### Operational Risks

**High Risk:**
- **SMSC Downtime:** Provider outages affecting message delivery
  - **Mitigation:** Automatic reconnection, status monitoring, application-level queuing

**Medium Risk:**
- **Rate Limiting by SMSC:** Throttling or blocking for excessive traffic
  - **Mitigation:** Built-in rate limiter interface, backoff on throttle errors

**Low Risk:**
- **Backward Compatibility:** Breaking changes in library updates
  - **Mitigation:** Semantic versioning, deprecation notices, changelog

---

## Assumptions

### Technical Assumptions

1. **SMSC Compliance:** SMSCs follow SMPP 3.4 specification reasonably closely
2. **Network Reliability:** TCP connections are generally stable for minutes to hours
3. **Go Runtime:** Go runtime garbage collector handles concurrent goroutines efficiently
4. **Character Sets:** Most SMS traffic uses GSM7 or UCS2 encodings

### Business Assumptions

1. **Use Case:** Primary use is transactional SMS in microservices architectures
2. **Scale:** Individual connections handle 10-1000 messages/second
3. **Reliability:** Automatic reconnection is preferred over manual intervention
4. **Integration:** Applications using this library handle persistence and queuing

---

## Dependencies

### External Dependencies

**Direct Dependencies:**
```
golang.org/x/text   v0.10.0  - Character encoding
golang.org/x/time   v0.3.0   - Rate limiting interface
golang.org/x/net    v0.11.0  - Network utilities
github.com/urfave/cli v1.22.14 - CLI framework (cmd/ only)
```

**Indirect Dependencies:**
- `github.com/cpuguy83/go-md2man/v2` (via urfave/cli)
- `github.com/russross/blackfriday/v2` (via urfave/cli)

### Infrastructure Dependencies

**Development:**
- Go toolchain 1.18+
- Git for version control
- Travis CI for automated testing

**Runtime:**
- SMSC endpoint (TCP port 2775 or TLS port 2776)
- Network connectivity to SMSC
- Optional: TLS certificates for secure connections

**Testing:**
- Built-in smpptest package (no external SMSC needed for unit tests)
- Optional: Real SMSC or SMPPSim for integration testing

---

## Timeline and Milestones

### Historical Milestones (Completed)

✅ **Phase 1 - Core Protocol (Completed)**
- PDU encoding/decoding
- Basic Transmitter/Receiver/Transceiver
- Character encodings (GSM7, Latin1, UCS2)

✅ **Phase 2 - Production Hardening (Completed)**
- Automatic reconnection
- Long message handling
- EnquireLink keepalive
- Error handling and status codes

✅ **Phase 3 - Developer Experience (Completed)**
- CLI tools (sms, smsapid)
- Comprehensive documentation
- Example code
- Test server (smpptest)

### Current State

**Status:** Production-ready, maintenance mode

**Recent Updates:**
- Enhanced documentation (100+ pages)
- Portuguese documentation added
- Code refactoring for maintainability

### Future Roadmap (Optional)

**Potential Enhancements:**
- DataSM support (low priority - rarely used)
- Enhanced observability hooks
- Performance optimizations
- Additional character encodings (if requested)

---

## Success Metrics

### Adoption Metrics

- **Internal Usage:** Primary goal is internal microservices
- **Stability:** Zero critical bugs in production
- **Performance:** Handle production SMS volumes without issues

### Quality Metrics

- **Test Coverage:** Comprehensive unit and integration tests
- **Documentation:** Complete API docs and examples
- **Code Quality:** Idiomatic Go, passes go vet and golint

### Operational Metrics (Application Responsibility)

Applications using go-smpp should track:
- Messages sent/received per second
- Connection uptime percentage
- Reconnection frequency
- Delivery receipt success rate
- Error rates by SMPP status code

---

## Conclusion

The go-smpp project successfully delivers a production-ready SMPP 3.4 library for Go that balances protocol compliance, reliability, and developer experience. The clear scope boundaries ensure the library remains focused on its core mission while allowing applications to implement higher-level features like persistence, queuing, and business logic.

---

**Document Version:** 1.0
**Last Updated:** 2025-10-31
**Next Review:** Quarterly or when major architectural changes are proposed
