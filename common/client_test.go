package common

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestExtractHeadersDecodesMIMEEncodedSubject(t *testing.T) {
	msg := &gmail.Message{Payload: &gmail.MessagePart{Headers: []*gmail.MessagePartHeader{
		{Name: "Subject", Value: "=?UTF-8?Q?Automation_Issue_=E2=80=93_TestLauncher?="},
	}}}

	got := ExtractHeaders(msg)["subject"]
	const want = "Automation Issue – TestLauncher"
	if got != want {
		t.Fatalf("ExtractHeaders() subject = %q, want %q", got, want)
	}
}

// IsInternalAddress decides who a reply is addressed to: an address wrongly
// judged internal is skipped as a recipient, and one wrongly judged external
// can become the recipient. Both failures send the reply to the wrong person,
// so the boundary cases are pinned here rather than left to inspection.
func TestIsInternalAddress(t *testing.T) {
	cases := []struct {
		name string
		addr string
		want bool
	}{
		{"bare blue.cc", "manny@blue.cc", true},
		{"bare blue.app", "manny@blue.app", true},
		{"display name form", "Manny from Blue <manny@blue.app>", true},
		{"mixed case", "Manny@Blue.APP", true},
		{"surrounding whitespace", "  manny@blue.cc  ", true},
		{"help alias", "help@blue.cc", true},

		// The in-app Feedback Form bot. Its address ends in blue.cc but sits on
		// a subdomain, and it carries real customer mail with Reply-To set to
		// the customer. Treating it as internal would strand those replies.
		{"feedback form bot", "Feedback Form <notifications@automations.blue.cc>", false},
		{"other subdomain", "x@mail.blue.app", false},

		// A substring test would have called these ours.
		{"lookalike domain", "someone@blue.cc.attacker.com", false},
		{"domain in local part", "blue.cc@gmail.com", false},
		{"customer mentioning us", "\"blue.cc support\" <person@example.com>", false},

		{"plain customer", "design@dickeyboats.com", false},
		{"empty", "", false},
		{"no at sign", "not-an-address", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsInternalAddress(tc.addr); got != tc.want {
				t.Errorf("IsInternalAddress(%q) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

func TestAddressDomain(t *testing.T) {
	cases := []struct{ in, want string }{
		{"manny@blue.app", "blue.app"},
		{"Manny from Blue <manny@blue.app>", "blue.app"},
		{"MANNY@BLUE.APP", "blue.app"},
		{"notifications@automations.blue.cc", "automations.blue.cc"},
		{"weird", ""},
		{"", ""},
	}

	for _, tc := range cases {
		if got := addressDomain(tc.in); got != tc.want {
			t.Errorf("addressDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
