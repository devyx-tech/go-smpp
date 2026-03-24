# Preocupações Técnicas

**Data de Análise:** 2026-03-24

## Tech Debt

### PDUs não implementados (5 tipos)

- **Localização:** `smpp/pdu/codec.go:180-212`, `smpp/pdu/factory.go:33-91`
- **Impacto:** Clientes que recebem AlertNotification, CancelSM, ReplaceSM, DataSM ou Outbind do SMSC receberão erro "PDU not implemented". Isso pode causar desconexões inesperadas.
- **Risco:** Médio — estes PDUs são menos comuns, mas SMSCs reais podem enviá-los
- **Abordagem de correção:** Para cada PDU, defina a struct com campos conforme spec SMPP 3.4, adicione constructor `newXxx(hdr)` + `NewXxx()` em `smpp/pdu/types.go`, e adicione o case em `Decode()` (`smpp/pdu/codec.go`) e `CreatePDU()` (`smpp/pdu/factory.go`)
- **Esforço:** Pequeno por PDU (cada um segue o mesmo padrão)

### HACK na serialização de UDH

- **Localização:** `smpp/pdu/codec.go:85-88`
- **Impacto:** A serialização pula campos UDH se não encontrados no PDU usando string comparison hardcoded (`"gsm_sms_ud.udh.len"`, `"gsm_sms_ud.udh"`). Isso funciona, mas é frágil.
- **Risco:** Baixo — funciona corretamente, mas quebra se os nomes dos campos mudarem
- **Abordagem de correção:** Usar um flag ou tipo de campo para indicar campos opcionais em vez de string comparison
- **Esforço:** Pequeno

### Validação de UnbindResp ausente no Close()

- **Localização:** `smpp/client.go:257`
- **Impacto:** Ao fechar a conexão, o client envia Unbind e lê a resposta do inbox, mas não valida se é realmente um UnbindResp. Qualquer PDU pendente no inbox seria consumido e descartado silenciosamente.
- **Risco:** Baixo — afeta apenas o shutdown graceful
- **Abordagem de correção:** Verificar se o PDU lido do inbox é `pdu.UnbindRespID`
- **Esforço:** Pequeno

### Validação de destinos em SubmitMulti incompleta

- **Localização:** `smpp/transmitter.go:449`
- **Impacto:** Apenas o número total de destinos é validado (max 254). Não há validação de formato de endereço ou tamanho de lista de distribuição.
- **Risco:** Baixo — a validação real é feita pelo SMSC
- **Abordagem de correção:** Adicionar validação de formato conforme spec SMPP 3.4
- **Esforço:** Pequeno

## Áreas Frágeis

- **`smpp/transmitter.go`** (586 linhas) — arquivo mais extenso do projeto. Concentra lógica de envio simples, envio longo, envio multi, e query. Modificar com cuidado pois os paths de Submit, SubmitLongMsg e SubmitMulti compartilham o mecanismo `do()` e o mapa `inflight`.

- **`smpp/pdu/pdufield/types.go`** (442 linhas) — contém muitos tipos de campos PDU. O decode de campos UDH (`UDHList`, `SM`) usa parsing manual de bytes que pode falhar silenciosamente com dados malformados.

- **`smpp/client.go:132-188`** — o loop de `Bind()` usa `goto retry` para reconexão. A lógica de cleanup (close `eli` channel, close conn, sleep) é sensível à ordem de operações.

## Segurança

- **Autenticação sem rate limiting:** O servidor (`smpp/server.go`) não limita tentativas de bind. Um atacante pode fazer brute-force de credenciais. Para produção, implemente rate limiting no `AuthRequestHandlerFunc`.

- **`rand.Seed` deprecated:** `smpp/server.go:65` usa `rand.Seed()` que é deprecated desde Go 1.20. Use `rand.New(rand.NewSource(...))` em vez disso.

## Performance

- **Mapa inflight com mutex:** `smpp/transmitter.go:57-58` — o mapa `inflight` é protegido por `sync.Mutex`, o que pode ser gargalo em cenários de altíssima vazão. Considerar `sync.Map` ou sharding para >10k msg/s.

- **Backoff exponencial com fator `e`:** O fator `math.E` (~2.718) resulta em delays que crescem rapidamente: 1s → 2.7s → 7.4s → 20s → 54s → 120s (max). Para cenários que exigem reconexão rápida, use `BindInterval` com valor fixo.

## TODOs e FIXMEs

- `smpp/client.go:257`: `TODO: validate UnbindResp`
- `smpp/pdu/codec.go:85`: `HACK: Skipping serialisation of UDH if it's not found in PDU`
- `smpp/pdu/codec.go:180`: `TODO: Implement AlertNotification`
- `smpp/pdu/codec.go:186`: `TODO: Implement CancelSM`
- `smpp/pdu/codec.go:188`: `TODO: Implement CancelSMResp`
- `smpp/pdu/codec.go:190`: `TODO: Implement DataSM`
- `smpp/pdu/codec.go:192`: `TODO: Implement DataSMResp`
- `smpp/pdu/codec.go:204`: `TODO: Implement Outbind`
- `smpp/pdu/codec.go:210`: `TODO: Implement ReplaceSM`
- `smpp/pdu/codec.go:212`: `TODO: Implement ReplaceSMResp`
- `smpp/server.go:87`: `TODO: Make sure Read(), Write() and Close() are working as expected`
- `smpp/transmitter.go:449`: `TODO: Validate numbers and lists according to size`
- `smpp/pdu/pdutext/doc.go:17`: `TODO: Fix this` (codec UCS2/Latin1)

## Melhorias Recomendadas

1. **Implementar PDUs faltantes** (AlertNotification, CancelSM, DataSM, Outbind, ReplaceSM) — Impacto: Médio — Esforço: Pequeno por PDU
2. **Substituir `rand.Seed` deprecated** — Impacto: Baixo — Esforço: Pequeno
3. **Adicionar rate limiting no auth do servidor** — Impacto: Alto (segurança) — Esforço: Pequeno
4. **Extrair `SubmitLongMsg` para método privado reutilizável** — reduzir `transmitter.go` — Impacto: Médio — Esforço: Médio
5. **Adicionar linter** (`.golangci.yml`) — Impacto: Médio (qualidade de código) — Esforço: Pequeno

---

*Análise de preocupações: 2026-03-24*
