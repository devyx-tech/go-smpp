# Contributing Guide - go-smpp

**Purpose:** Development workflow, testing requirements, build process, and CI/CD configuration.

---

## Branch Strategy

**Main Branch:** `master`

**Branch Naming:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring

**Example:**
```bash
git checkout -b feature/add-data-sm-support
git checkout -b fix/encoding-issue-gsm7
git checkout -b docs/update-readme
```

---

## Development Setup

### Prerequisites

- Go 1.18 or higher
- Git
- (Optional) SMPPSim or real SMSC for integration testing

### Clone and Build

```bash
git clone https://github.com/devyx-tech/go-smpp.git
cd go-smpp

# Download dependencies
go mod download

# Build library
go build ./...

# Run tests
go test ./...
```

---

## Testing Requirements

### Unit Tests

**Location:** `*_test.go` files alongside implementation

**Run:**
```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...

# Specific package
go test ./smpp/pdu/pdutext/
```

**Coverage Goal:** Aim for >70% coverage on new code

### Integration Tests

**Pattern:** Use `smpptest` package for mock SMSC

```go
func TestIntegration(t *testing.T) {
    srv := smpptest.NewUnstartedServer(mockHandler)
    srv.Start()
    defer srv.Close()

    tx := &smpp.Transmitter{Addr: srv.Addr}
    // Test with real client against mock server
}
```

### Race Detection

```bash
go test -race ./...
```

**Requirement:** All tests must pass with race detector

### Table-Driven Tests

**Preferred Pattern:**
```go
tests := []struct {
    name string
    input string
    want []byte
    err error
}{
    {"case1", "input1", []byte{0x00}, nil},
    {"case2", "input2", []byte{0x01}, nil},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := fn(tt.input)
        // Assertions
    })
}
```

---

## Code Style

### Formatting

```bash
# Format code
gofmt -w .

# Or use goimports (auto-adds imports)
goimports -w .
```

**Requirement:** All code must be gofmt'd before commit

### Linting

```bash
# Static analysis
go vet ./...

# (Optional) golangci-lint
golangci-lint run
```

### Naming Conventions

- **Packages:** lowercase, singular (`smpp`, `pdu`, not `SMPPs`, `pdus`)
- **Types:** PascalCase (`Transmitter`, not `transmitter`)
- **Functions:** PascalCase for exported, camelCase for internal
- **No "Get" prefix:** `Header()`, not `GetHeader()`
- **Constants:** PascalCase with descriptive names

---

## Pull Request Process

### 1. Create Feature Branch

```bash
git checkout master
git pull origin master
git checkout -b feature/my-feature
```

### 2. Make Changes

- Write code following style guidelines
- Add tests for new functionality
- Update documentation if needed
- Ensure all tests pass

### 3. Commit

```bash
git add .
git commit -m "feat: add support for DataSM PDU"
```

**Commit Message Format:**
```
type: short description

Longer explanation if needed

Fixes #issue_number
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

### 4. Push and Create PR

```bash
git push origin feature/my-feature
```

Then create Pull Request on GitHub.

### 5. PR Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Code is formatted (`gofmt`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Documentation updated
- [ ] Commit messages are clear
- [ ] No breaking changes (or documented in PR)

---

## Build Process

### Local Build

```bash
# Build library
go build ./...

# Build CLI tools
go build -o sms ./cmd/sms
go build -o smsapid ./cmd/smsapid

# Install CLI tools globally
go install ./cmd/sms
go install ./cmd/smsapid
```

### Cross-Platform Build

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o sms-linux ./cmd/sms

# Windows
GOOS=windows GOARCH=amd64 go build -o sms.exe ./cmd/sms

# macOS
GOOS=darwin GOARCH=amd64 go build -o sms-darwin ./cmd/sms
```

---

## CI/CD Configuration

### Travis CI

**File:** `.travis.yml`

```yaml
language: go
go:
  - "1.9.4"  # Update to match minimum supported version
install:
  - go get -d -v ./...
  - go get -d -v golang.org/x/tools/cmd/cover
  - go get golang.org/x/time/rate
script:
  - go test -v -cover ./...
```

**Runs on:**
- Every commit to any branch
- All Pull Requests

**Build Status:** Check badge in README.md

---

## Adding New Features

### Adding a New PDU Type

1. Create `smpp/pdu/my_pdu.go`
2. Define struct:
   ```go
   type MyPDU struct {
       *Codec
   }

   func NewMyPDU() *MyPDU {
       return &MyPDU{
           Codec: &Codec{
               H: &Header{ID: MyPDUID},
               L: pdufield.List{/* fields */},
           },
       }
   }

   func (p *MyPDU) FieldList() pdufield.List {
       return p.L
   }
   ```
3. Add constant: `const MyPDUID = 0x...`
4. Update factory if needed
5. Add tests in `my_pdu_test.go`
6. Update documentation

### Adding a New Text Encoding

1. Create `smpp/pdu/pdutext/myencoding.go`
2. Implement `Codec` interface
3. Add tests in `myencoding_test.go`
4. Document character set and limitations
5. Update docs/features.md

### Modifying Existing Behavior

1. Check if change is breaking
2. Add deprecation notice if removing functionality
3. Update tests
4. Update documentation
5. Add migration guide if needed

---

## Documentation Standards

### Code Documentation

**Package docs:**
```go
// Package smpp implements SMPP 3.4 protocol for Go.
//
// This package provides Transmitter, Receiver, and Transceiver
// clients for sending and receiving SMS via SMSC connections.
//
// Example:
//   tx := &smpp.Transmitter{Addr: "localhost:2775", ...}
//   tx.Bind()
//   tx.Submit(&smpp.ShortMessage{...})
package smpp
```

**Type docs:**
```go
// Transmitter is an SMPP client for sending SMS messages.
// It maintains a persistent connection with automatic reconnection.
type Transmitter struct {
    // Addr is the SMSC address (e.g., "smsc.example.com:2775")
    Addr string
    // ...
}
```

**Function docs:**
```go
// Submit sends a single SMS message to the SMSC.
// Returns SubmitSMResp with MessageID on success, or error.
//
// If the connection is down, returns ErrNotConnected.
// If SMSC returns error status, returns pdu.Status error.
func (t *Transmitter) Submit(sm *ShortMessage) (*SubmitSMResp, error)
```

### Markdown Documentation

- Use clear headings (## for sections, ### for subsections)
- Include code examples
- Link to related documentation
- Keep language concise and technical

---

## Release Process

### Versioning

**Semantic Versioning:** `MAJOR.MINOR.PATCH`

- **MAJOR:** Breaking changes
- **MINOR:** New features (backward compatible)
- **PATCH:** Bug fixes

### Creating a Release

1. Update CHANGELOG.md
2. Tag release:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```
3. GitHub will automatically create release
4. Users update with:
   ```bash
   go get github.com/devyx-tech/go-smpp@v1.2.3
   ```

---

## Getting Help

**Issues:** https://github.com/devyx-tech/go-smpp/issues

**Discussions:** GitHub Discussions (if enabled)

**Documentation:** See `/docs` folder and this `/specs/technical` folder

---

**Last Updated:** 2025-10-31
