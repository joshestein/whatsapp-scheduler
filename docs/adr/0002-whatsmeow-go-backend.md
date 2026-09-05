# Use whatsmeow, so the backend is Go

Three unofficial libraries fit ADR-0001. whatsapp-web.js (Node) drives a headless Chromium through Puppeteer. Baileys (Node) and whatsmeow (Go) speak the WhatsApp Web WebSocket protocol directly, no browser.

We picked whatsmeow. It is the engine under the mautrix-whatsapp bridge, so protocol breakage gets fixed fast by people who run it in production. It has built-in auto-reconnect and a typed `LoggedOut` event, which map directly onto the connected / disconnected / logged out Session states in CONTEXT.md. Its `sqlstore` persists the session in SQLite and can wrap the app's own `*sql.DB`, so one file holds session and messages. The app ships as one static binary, which makes the later move to a box trivial.

Rejected: whatsapp-web.js, because a suspended Chromium after laptop sleep often hangs silently and needs a full client restart. Baileys, because reconnect is manual, the session store is a directory of JSON files, and version 7 is still a release candidate. Ban risk is the same for all three. It tracks sending behaviour, not library.

Costs: whatsmeow is pre-1.0 with pseudo-versions and API churn between commits, so pin the exact version in go.mod and read the changelog before bumping. Documentation is godoc, the `mdtest` example, and the mautrix-whatsapp source. Messages are built as protobufs (`waE2E.Message`). The frontend stays server-rendered plus htmx, via `html/template`.
