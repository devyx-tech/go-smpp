# Stack Tecnológica

**Data de Análise:** 2026-03-24

## Linguagem

**Primária:**
- Go 1.18 — toda a base de código

## Runtime

- Go toolchain 1.18+
- Package Manager: Go Modules
- Lockfile: `go.sum` presente

## Frameworks

**Core:**
- Nenhum framework — biblioteca pura implementando o protocolo SMPP 3.4 sobre TCP

**Testes:**
- `testing` (stdlib) — framework de testes padrão do Go
- `smpp/smpptest` — servidor SMPP fake interno para testes de integração

**CLI:**
- `github.com/urfave/cli` v1.22.14 — CLI framework para a ferramenta `cmd/sms/`

## Dependências Chave

**Críticas (afetam arquitetura):**
- `golang.org/x/net` v0.11.0 — fornece `context` usado no `RateLimiter` e conexões
- `golang.org/x/text` v0.10.0 — encoding/decoding de texto (Latin1, ISO-8859-5, UCS2) via `transform` e `encoding`
- `golang.org/x/time` v0.3.0 — `rate.Limiter` usado como `RateLimiter` no Transmitter/Transceiver

**Indiretas:**
- `github.com/cpuguy83/go-md2man/v2` v2.0.2 — dependência indireta do `urfave/cli`
- `github.com/russross/blackfriday/v2` v2.1.0 — dependência indireta do `urfave/cli`

## Banco de Dados

Não aplicável — biblioteca de protocolo de rede, sem persistência.

## Configuração

**Variáveis de ambiente (CLI apenas):**
- `SMPP_USER` — username SMPP (fallback quando `--user` não é passado na CLI)
- `SMPP_PASSWD` — password SMPP (fallback quando `--passwd` não é passado na CLI)

**Arquivos de configuração:**
- `go.mod` — definição do módulo e dependências
- Nenhum arquivo de configuração de aplicação — toda configuração é feita via structs Go (`Transmitter`, `Receiver`, `Transceiver`)

## Infraestrutura e Deploy

- Plataforma: biblioteca — distribuída como módulo Go, não deployada diretamente
- Containers: Não aplicável
- CI/CD: GitHub Actions — `.github/workflows/claude.yml`, `.github/workflows/claude-code-review.yml`

## Decisões Arquiteturais

- **Go Modules em vez de vendor** — módulo publicado em `github.com/devyx-tech/go-smpp`, consumido via `go get`
- **Sem dependência em framework de rede** — usa `net` e `bufio` da stdlib para máximo controle sobre o protocolo binário
- **`golang.org/x/text` para encoding** — delega encoding ISO/UCS2 para a biblioteca oficial. GSM7 é implementado internamente em `smpp/encoding/gsm7.go` pois não existe na stdlib
- **`urfave/cli` apenas no cmd/** — a biblioteca core (`smpp/`) não depende do CLI framework

---

*Análise de stack: 2026-03-24*
