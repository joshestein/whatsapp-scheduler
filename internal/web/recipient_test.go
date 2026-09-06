package web

import "testing"

func TestNormaliseRecipient(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"2712345", "2712345@s.whatsapp.net", true},
		{"+27 123 456", "27123456@s.whatsapp.net", true},
		{"2712345@s.whatsapp.net", "2712345@s.whatsapp.net", true},
		{"120363000000000000@g.us", "120363000000000000@g.us", true},
		{"123456789012345@lid", "123456789012345@lid", true},
		{"Jo", "", false},
		{"", "", false},
		{"abc@s.whatsapp.net", "", false},
		{"27123456@example.com", "", false},
	}
	for _, tc := range cases {
		got, err := normaliseRecipient(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("normalizeRecipient(%q) = %q, %v; want %q, ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}
