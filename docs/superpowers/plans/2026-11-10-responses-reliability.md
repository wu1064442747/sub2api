# Responses reliability repair

Goal: repair the four diagnosed failures without inventing tool call IDs or replaying completed tools.

Authorized approach: user asked to implement the previously described fixes. Keep existing platform and failover design.

- [x] Add regressions for Codex Desktop page-context injections (agent.codex_apps_open_page and selected_page snapshot), valid tool history, ambiguous/malformed messages, idempotence.
- [x] Normalize only recognized missing-ID page-context injections into user messages before function-output validation; keep ordinary tool results strict.
- [x] Add regression for tool requests resolving to GPT-6.1 on CC-only accounts, including account model aliases. Skip incompatible selected accounts before acquiring account slots or forwarding; preserve plain chat and other providers.
- [x] Configure existing first-output protection (standard 60s, high effort 120s), retain bounded one-time failover; inspect ingress path for verified defects before changing upload deadlines.
- [x] Run targeted and package tests, build versioned image preserving the current frontend, back up production config/database, deploy with rollback and health/smoke checks.

Constraints: preserve pricing/user data and pre-existing research-evidence; no random call IDs; do not disable request validation or assume HTTP 200 implies completion. Record any limitations in final report.

Verification: initial regressions failed, then passed; independent review found input additional_tools and mixed bootstrap gaps, both reproduced RED then fixed GREEN. Handler/service/httputil/middleware suites pass. Frontend locale tests, typecheck and Vite build pass; Linux embed binary builds. Production direct/public 440,037-byte zstd ingress test completed in 24/34ms (local missing-model 400, no upstream generation).

Deployment: server-local image `sub2api:responses-fix-20261110`, source a8b55172a, binary SHA256 4214822ff24daee80c6e713096b39b5dd25e353bd080d1daf61d99914f4cf907. Runtime Compose and Dokploy stored composeFile both updated. Backup: /home/ubuntu/sub2api-responses-fix-20261110 (database, runtime config, Dokploy record). First deployment rolled back on transient public 404 after container health; bounded public-route readiness polling added, second deployment passed. Container cda277e01311 reports healthy with zero restarts and version 0.2.20-responses.1.

Live checks: ordinary missing-call-id tool result rejected400; observed page-context injections plus forced synthetic GPT-6.1 tool returned response.completed and repair_health_check in 3.386s, no tool actually executed. Post-deployment 440037-byte zstd public upload decoded and rejected locally for missing model in30ms. Actual production logs show page-context normalization and CC-only account128 skipped. Original interrupted upload was not reproducible; no claim of permanent network repair. No payment or pricing changes. Source remains on local fix/responses-tool-routing branch; image is local to the production host, not published to registry.
