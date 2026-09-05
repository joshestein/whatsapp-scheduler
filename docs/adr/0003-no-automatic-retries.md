# Never retry a Send automatically

A Send can fail after the message has already left: the library times out, the socket drops before the ack. From the outside this looks the same as a real rejection. A retry then sends the message twice.

For personal messages to real people, a duplicate is worse than a miss, and every extra send is a ban signal. So a failed Send goes straight to `failed`, the UI notifies, and the user checks WhatsApp and reschedules by hand if needed. Same rule for a process crash mid-send: on startup, `sending` rows become `failed` with reason "unknown outcome".
