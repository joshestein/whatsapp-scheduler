# Context / Glossary

Domain language for the WhatsApp Scheduler. One word, one meaning.

## Terms

**Scheduled Message** — a text message, a target Contact, and a send time. It waits until its send time.

**Session** — the authenticated link to the user's own WhatsApp account. Made once by QR scan. Must stay alive so messages send without a new scan. Has one of three states, see below.

**Contact** — a person or group in the user's WhatsApp who can receive a message, identified by JID.

**Send** — the act of delivering a Scheduled Message through the live Session.

**Command Channel** — the user's self-chat ("Message yourself" in WhatsApp). The app watches it and acts on **Commands** sent there. Only the user can write to it; the app also checks `IsFromMe`.

**Command** — a message in the Command Channel that starts with `/verb`. v1 has one verb, `/schedule`, which creates a Scheduled Message. The verb space is open for later (`/list`, `/cancel`, `/failed`), but the dashboard stays the source of truth for the list, cancel, and missed/failed; chat verbs are added only where opening the dashboard is real friction. A message that does not start with `/` is not a Command and is ignored.

**Grace Window** — 30 minutes after the send time. Inside it, a late Send is acceptable. Past it, the message becomes Missed.

**Missed** — not attempted inside its Grace Window (machine asleep, Session down). Never sent. Timing failure.

**Failed** — attempted, but did not complete. Two causes: WhatsApp rejected it (invalid recipient, blocked), or the process died mid-send and the outcome is unknown. Never retried (ADR-0003). The user checks WhatsApp and decides.

## Message states

```
pending ──(due, inside Grace Window)──▶ sending ──ok────────────▶ sent
   │                                       └──rejected/unknown──▶ failed
   └──(Grace Window passed)──────────────────────────────────────▶ missed
```

- **pending** — waiting for its send time.
- **sending** — claimed by the scheduler, in flight. Exists for crash recovery only: on startup, every `sending` row becomes `failed` with reason "unknown outcome".
- **sent**, **missed**, **failed** — terminal. The UI notifies the user for missed and failed.

## Session states

- **connected** — socket open. Sends go through.
- **disconnected** — socket closed, credentials still valid. The app reconnects on its own. Pending messages wait. The Grace Window keeps running.
- **logged out** — credentials revoked (user unlinked the device, or WhatsApp forced it). The app does not reconnect. The UI shows a new QR code.

## Scheduler

A poll loop, not per-message timers. Every 60 seconds, one tick:

1. Check the Session is connected. If not, do nothing this tick.
2. Mark `pending` messages past their Grace Window as `missed`.
3. Send `pending` messages that are due, one at a time.

Ticks do not overlap. The next tick is scheduled after the current one finishes. The database is the single source of truth. This survives process restart and machine sleep: the next tick after wake finds the due messages. Timers cannot do this.

## Time

A send time is an instant plus the time zone it was created in. The browser supplies the zone. The UI accepts and shows the time in that creation zone, with the zone named (e.g. `Africa/Johannesburg`), the same on any viewing device. The instant is kept in UTC; the Scheduler and Grace Window act on the instant alone, so the zone never affects timing.

## Ban risk

Every unofficial client risks a permanent ban (ADR-0001). Risk tracks sending behaviour, not library. Rules: low volume, no retries, no bulk, only Contacts already in the chat list.
