# Read and act on Command Channel messages

Until now the app only sends. The `events.Message` handler is empty. To let the user schedule from chat instead of the dashboard, the app now reads the user's own self-chat ("Message yourself") and acts on `/schedule` Commands. Design in `docs/whatsapp_interface.md`.

We accept the inbound path because it is narrow: one chat that only the user can write to, and only messages that start with `/`. The dashboard stays the source of truth. Chat is a create-only shortcut.

Alternatives we rejected:

- **A dedicated group.** Keeps replies out of the notes space, but needs a group JID in config, and the app has no UI that shows one. A wrong JID sends replies with contact names into the wrong group. The `/` prefix already keeps Commands apart from notes, and the own JID is known from the Session, so self-chat needs no setup and cannot leak.
- **Natural-language parsing.** Needs an LLM or a heavy grammar, against the one-static-binary rule. A silent misread sends to the wrong person at the wrong time.

Reading is passive. Replies are self-to-self, a few a day, the lowest-risk send shape there is. The exception is the `expired` reject, which gets no reply: a reconnect can redeliver many old Commands at once, and a reply to each is a burst of sends, the bulk shape the ban rule avoids.

The design rests on whatsmeow behaviour a spike must confirm first. The main risk: a phone-sent self-chat message must reach the linked device as an `IsFromMe` `events.Message`. Confirmed for a group; the self-chat run is pending.
