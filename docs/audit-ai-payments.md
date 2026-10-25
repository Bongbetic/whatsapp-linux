# Meta-AI + payments audit (WL-4.8)

Date: 2026-10-26. Verdict: **both out of v1 scope**.

## Meta AI

- The reference Windows build surfaces Meta AI entry points; the
  multi-device companion protocol carries them as regular message
  flows plus client-side feature flags.
- whatsapp-linux implements no AI surface: no prompt UI, no model
  calls, no AI feature-flag handling. If a chat contains AI
  messages they render as plain text via the normal ingest path;
  nothing is sent to any AI endpoint by this client.
- Revisit only if protocol interop requires acknowledging AI
  stanzas (none observed at pinned whatsmeow).

## Payments

- Reference build contains payment rails (region-gated). The
  companion protocol exposes no payments API in pinned whatsmeow,
  and this client implements no wallet, no payment message
  composition, and no transaction handling.
- Inbound payment-request messages (if any) render as unsupported
  system rows; no funds can move through this client by design.

Both verdicts stand unless a Gate 4/8 interop row proves a missing
surface breaks daily driving — then scope, don't silently expand.
