# Technical Context Documentation - go-smpp

## Perfil de Contexto do Projeto

**Project Name:** go-smpp - SMPP 3.4 Library for Go

**Version:** Based on SMPP 3.4 Protocol Specification (Issue 1.2)

**Technology Stack:** Go 1.18+

**Primary Purpose:** Production-ready SMPP 3.4 (Short Message Peer-to-Peer Protocol) implementation for Go, enabling SMS sending and receiving through SMSC connections in microservices architectures.

**Team Structure:** Open-source project with internal usage focus

**Development Constraints:**
- Must maintain SMPP 3.4 protocol compliance
- Zero-downtime reconnection required
- Thread-safe concurrent operations
- Minimal external dependencies
- Cross-platform support (Linux, macOS, Windows)

---

## Camada 1: Contexto Central do Projeto

### Core Documentation

- [**Carta do Projeto (Project Charter)**](project_charter.md)
  - Project vision, success criteria, scope boundaries
  - Key stakeholders and technical constraints

- [**Registros de Decisões Arquiteturais (ADRs)**](adr/)
  - [ADR-001: Channel-Based Status Notifications](adr/001-channel-based-status.md)
  - [ADR-002: Automatic Reconnection with Exponential Backoff](adr/002-automatic-reconnection.md)
  - [ADR-003: Separate Client Types (Transmitter/Receiver/Transceiver)](adr/003-separate-client-types.md)
  - [ADR-004: In-Memory Long Message Merging](adr/004-in-memory-message-merging.md)
  - [ADR-005: Interface-Based Design for Extensibility](adr/005-interface-based-design.md)
  - [ADR-006: No Built-in Persistence](adr/006-no-persistence.md)

---

## Camada 2: Arquivos de Contexto Otimizados para IA

### AI-Optimized Development Guides

- [**Guia de Desenvolvimento com IA (CLAUDE.meta.md)**](CLAUDE.meta.md)
  - Code style patterns extracted from codebase
  - Testing approaches and conventions
  - Common patterns and anti-patterns
  - Performance considerations
  - SMPP-specific implementation gotchas

- [**Guia de Navegação da Base de Código (CODEBASE_GUIDE.md)**](CODEBASE_GUIDE.md)
  - Directory structure with purpose annotations
  - Key files and their roles
  - Data flow patterns
  - Integration points and dependencies
  - Deployment architecture

---

## Camada 3: Contexto Específico do Domínio

### Domain-Specific Documentation

- [**Documentação da Lógica de Negócio (BUSINESS_LOGIC.md)**](BUSINESS_LOGIC.md)
  - SMPP 3.4 protocol rules and validations
  - PDU structure and field requirements
  - Character encoding handling (GSM7, UCS2, Latin1, ISO-8859-5)
  - Long message concatenation logic
  - Delivery receipt workflows

- [**Especificações da API (API_SPECIFICATION.md)**](API_SPECIFICATION.md)
  - Public API surface (Transmitter, Receiver, Transceiver)
  - Method signatures and parameters
  - Error handling and status codes
  - Rate limiting interface
  - Middleware composition

---

## Camada 4: Contexto do Fluxo de Desenvolvimento

### Development Workflow Context

- [**Guia de Fluxo de Desenvolvimento (CONTRIBUTING.md)**](CONTRIBUTING.md)
  - Branch strategy and git workflow
  - Code review process
  - Testing requirements (unit, integration)
  - Build and deployment process
  - CI/CD configuration (Travis CI)

- [**Guia de Solução de Problemas (TROUBLESHOOTING.md)**](TROUBLESHOOTING.md)
  - Common connection issues
  - SMSC provider compatibility problems
  - Character encoding troubleshooting
  - Performance debugging
  - Network and timeout issues

- [**Desafios Arquiteturais (ARCHITECTURE_CHALLENGES.md)**](ARCHITECTURE_CHALLENGES.md)
  - Current limitations and known issues
  - Scalability considerations
  - SMSC provider quirks
  - Future improvement areas

---

## Quick Navigation

### For New Developers
1. Start with [Project Charter](project_charter.md) to understand the vision
2. Read [CODEBASE_GUIDE.md](CODEBASE_GUIDE.md) to navigate the code
3. Review [CLAUDE.meta.md](CLAUDE.meta.md) for development patterns
4. Check [CONTRIBUTING.md](CONTRIBUTING.md) for workflow

### For AI Assistants
1. Review [CLAUDE.meta.md](CLAUDE.meta.md) for code style and patterns
2. Consult [BUSINESS_LOGIC.md](BUSINESS_LOGIC.md) for SMPP protocol rules
3. Reference [API_SPECIFICATION.md](API_SPECIFICATION.md) for public interfaces
4. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues

### For Architects
1. Review [ADRs](adr/) for architectural decisions and rationale
2. Read [ARCHITECTURE_CHALLENGES.md](ARCHITECTURE_CHALLENGES.md) for current limitations
3. Consult [BUSINESS_LOGIC.md](BUSINESS_LOGIC.md) for protocol constraints

### For DevOps/SRE
1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for operational issues
2. Review [CONTRIBUTING.md](CONTRIBUTING.md) for deployment process
3. Reference [API_SPECIFICATION.md](API_SPECIFICATION.md) for monitoring integration points

---

## Related Documentation

This technical documentation complements the existing documentation in `/docs`:

- [docs/index.md](../../docs/index.md) - User-facing getting started guide
- [docs/stack.md](../../docs/stack.md) - Technology stack overview
- [docs/patterns.md](../../docs/patterns.md) - Design patterns used
- [docs/features.md](../../docs/features.md) - Feature documentation
- [docs/business-rules.md](../../docs/business-rules.md) - SMPP protocol rules
- [docs/integrations.md](../../docs/integrations.md) - Integration examples

---

## Document Maintenance

**Last Updated:** 2025-10-31

**Update Frequency:**
- Review ADRs when making architectural changes
- Update CLAUDE.meta.md when patterns evolve
- Refresh TROUBLESHOOTING.md as new issues are discovered
- Revise ARCHITECTURE_CHALLENGES.md quarterly

**Contributors:** See [AUTHORS](../../AUTHORS) and [CONTRIBUTORS](../../CONTRIBUTORS)

---

## Links and References

- **SMPP 3.4 Specification:** https://smpp.org/SMPP_v3_4_Issue1_2.pdf
- **GitHub Repository:** https://github.com/devyx-tech/go-smpp
- **Go Package Documentation:** Run `go doc github.com/devyx-tech/go-smpp/smpp`
- **Original smpp34 Project:** https://github.com/CodeMonkeyKevin/smpp34
