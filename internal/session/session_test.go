package session

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestDerive(t *testing.T) {
	pn := func(u string) types.JID { return types.NewJID(u, types.DefaultUserServer) }
	lid := func(u string) types.JID { return types.NewJID(u, types.HiddenUserServer) }
	me := pn("27000")

	all := map[types.JID]types.ContactInfo{
		pn("27001"): {FullName: "bob", PushName: "Bobby"}, // FullName wins
		pn("27002"): {PushName: "Alice"},                  // PushName fallback
		pn("27003"): {BusinessName: "Zed Ltd"},            // BusinessName fallback
		pn("27004"): {},                                   // nameless: drop
		lid("900"):  {PushName: "Carol"},                  // mapped, no PN row: becomes 27005
		lid("901"):  {PushName: "Bob (lid)"},              // mapped, PN row exists: merge, PN wins
		lid("902"):  {PushName: "Ghost"},                  // unmapped: drop
		me:          {FullName: "Josh"},                   // own number: Me, not a person
	}
	lidToPN := map[types.JID]types.JID{lid("900"): pn("27005"), lid("901"): pn("27001")}
	groups := []*types.GroupInfo{
		{JID: types.NewJID("1", types.GroupServer), GroupName: types.GroupName{Name: "dev"}},
		{JID: types.NewJID("2", types.GroupServer)},
	}

	got := derive(me, all, lidToPN, groups)

	want := []Contact{
		{"27000@s.whatsapp.net", false, "Me", "27000"},
		{"27002@s.whatsapp.net", false, "Alice", "27002"},
		{"27001@s.whatsapp.net", false, "bob", "27001"},
		{"27005@s.whatsapp.net", false, "Carol", "27005"},
		{"1@g.us", true, "dev", ""},
		{"2@g.us", true, "Unnamed group", ""},
		{"27003@s.whatsapp.net", false, "Zed Ltd", "27003"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d contacts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSearch(t *testing.T) {
	cases := []struct {
		c    Contact
		want string
	}{
		{Contact{Primary: "Bob", Secondary: "27001"}, "bob 27001"},
		{Contact{Primary: "Me", Secondary: "27000"}, "me 27000"},
		{Contact{IsGroup: true, Primary: "Dev Chat"}, "dev chat"},
	}
	for _, tc := range cases {
		if got := tc.c.Search(); got != tc.want {
			t.Errorf("Search(%+v) = %q, want %q", tc.c, got, tc.want)
		}
	}
}
