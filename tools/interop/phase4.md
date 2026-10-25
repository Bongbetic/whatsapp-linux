# Phase 4 interop notes — Gate 4 (WL-4.1..WL-4.8)

## WL-4.1 groups

- [ ] Create group with 3 members; rename; set topic; add/remove/
      promote/demote; invite link show + reset; join via link.
- [ ] Approval queue: listed read-only (manage API lands with fork
      if interop needs it).

## WL-4.2 communities

- [ ] View subgroups; link/unlink; announcement send.

## WL-4.3 status

- [ ] Post text status; appears on phone; rings list shows
      contacts' statuses; reply lands in DM.

## WL-4.4 channels

- [ ] Follow/unfollow; muted state round-trips; feed shows updates.
- [ ] Directory search + posting-as-channel: eligibility-scoped,
      verdict at Phase 8.

## WL-4.5 business

- [ ] Profile dialog shows address/categories; catalog/product
      messages + cart: eligibility-scoped, verdict at Phase 8.

## WL-4.6 privacy & safety

- [ ] Each privacy row writes server-side and re-reads correctly.
- [ ] Block/unblock round-trips; blocklist mirrors contacts rows.
- [ ] Disappearing default + per-chat timer.

## WL-4.7 fork gaps roundup (§7 gaps minus calls)

| Gap | Status |
|---|---|
| call place/accept + media keys | FORK at WL-5.1 |
| call links create/join | FORK at WL-5.1 |
| status posting parity | DONE upstream (`SendMessage` to status broadcast, WL-4.3) |
| community create | upstream `CreateGroup` with parent context — verify at Gate 4, else fork |
| channel directory search | upstream has no directory API — documented gap, Phase 8 verdict |
| catalog list beyond `GetOrderDetails` | documented gap, Phase 8 verdict |
| message edit window errors | surfaced from send errors, no fork |
| LID/PN mapping edges | upstream `StoreLIDPNMapping` exists, no fork |
| passkey pairing | exposed only if phone offers it (no UI until observed) |

## WL-4.8 Meta-AI + payments audit

See `docs/audit-ai-payments.md`: both out of v1 scope, client
sends nothing to either surface.
