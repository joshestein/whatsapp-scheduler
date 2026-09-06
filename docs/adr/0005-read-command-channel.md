# Read and act on Command Channel messages

Until now the app only sends. The `events.Message` handler is empty. To let the user schedule from chat instead of the dashboard, the app now reads one configured group (the Command Channel) and acts on `/schedule` Commands. Design in `docs/whatsapp_interface.md`.

We accept the inbound path because it is narrow: one group, set by `COMMAND_GROUP_JID`, and only messages from the user's own number (`IsFromMe`). The dashboard stays the source of truth. Chat is a create-only shortcut.

Alternatives we rejected:

- **Self-chat instead of a group.** The self-chat is the user's notes space. Bot replies would clutter it.
- **Natural-language parsing.** Needs an LLM or a heavy grammar, against the one-static-binary rule. A silent misread sends to the wrong person at the wrong time.

Reading is passive and carries low ban risk. Replies go to one group, a few a day. The exception is the `expired` reject, which gets no reply: a reconnect can redeliver many old Commands at once, and a reply to each is a burst of self-sends, the bulk shape the ban rule avoids.

The design rests on whatsmeow behaviour a spike must confirm first. The main risk: a phone-sent group message must reach the linked device as an `IsFromMe` `events.Message`. If not, the feature stops.
