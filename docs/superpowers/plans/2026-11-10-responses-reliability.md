# Responses reliability repair

Goal: repair the four diagnosed failures without inventing tool call IDs or replaying completed tools.

Authorized approach: user asked to implement the previously described fixes. Keep existing platform and failover design.

- [x] Add regressions for Codex Desktop page-context injections (agent.codex_apps_open_page and selected_page snapshot), valid tool history, ambiguous/malformed messages, idempotence.
- [x] Normalize only recognized missing-ID page-context injections into user messages before function-output validation; keep ordinary tool results strict.
- [x] Add regression for tool requests resolving to GPT-6.1 on CC-only accounts, including account model aliases. Skip incompatible selected accounts before acquiring account slots or forwarding; preserve plain chat and other providers.
- [ ] Configure existing first-output protection (standard 60s, high effort 120s), retain bounded one-time failover; inspect ingress path for verified defects before changing upload deadlines.
- [ ] Run targeted and package tests, build versioned image preserving the current frontend, back up production config/database, deploy with rollback and health/smoke checks.

Constraints: preserve pricing/user data and pre-existing research-evidence; no random call IDs; do not disable request validation or assume HTTP 200 implies completion. Record any limitations in final report.

Verification: initial regressions failed, then passed; independent review found input additional_tools and mixed bootstrap gaps, both reproduced RED then fixed GREEN. Handler/service/httputil/middleware suites pass. Frontend locale tests, typecheck and Vite build pass; Linux embed binary builds. Production direct/public 440,037-byte zstd ingress test completed in 24/34ms (local missing-model 400, no upstream generation).
