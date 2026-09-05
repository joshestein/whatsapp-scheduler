# Use unofficial WhatsApp Web automation, not the official Business Cloud API

We schedule messages from the user's own personal WhatsApp number. The official Business Cloud API cannot do this: it needs a business number and only allows free-text sends inside a 24-hour user-initiated window (pre-approved templates otherwise). So we drive a real WhatsApp Web session with an unofficial library (QR login once, keep the session alive).

This violates WhatsApp Terms of Service and carries a real risk that the number is banned. We accept that risk because the tool is self-hosted and personal, for one user's own number, not a marketed multi-tenant product. If this ever becomes a sold product, this decision must be revisited: the ban risk concentrates on paying customers' numbers and the ToS problem becomes a legal one.
