# Store a send time as a UTC instant plus its creation zone name

A send time must do two jobs: fire at the exact instant, and show the user the wall-clock they meant. The old rule stored only the UTC instant and used the machine's local zone for input and display. Move the backend to a box in another zone and every time shifts with the box.

So we store two columns: the UTC unix instant (unchanged) and the IANA zone name the message was created in, for example `Africa/Johannesburg`. The browser supplies the name on submit. The server parses input and renders the list in that zone, name shown. The time reads the same on any device.

Alternatives we rejected:

- **Offset in the timestamp** (`...+02:00`). An offset is not a zone. It cannot render the name, and it shows the wrong wall-clock once DST shifts between creation and send.
- **RFC 9557** (`...+02:00[Africa/Johannesburg]`). Go's stdlib does not parse the suffix, and the Scheduler loses cheap integer comparison.
- **RFC 3339 text for the instant.** Still needs a zone column, and adds string parsing to the tick loop for nothing.
- **Display in the viewer's zone.** The number changes between devices, which reads as "my send time moved".

The Scheduler and Grace Window use the instant alone, so the zone never affects timing. A missing or invalid zone name (JavaScript off, junk) falls back to the machine's local zone. The binary imports `time/tzdata` so lookups do not depend on the box having a zone database.
