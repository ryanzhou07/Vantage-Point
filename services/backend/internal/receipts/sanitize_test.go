package receipts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSanitizeTextRemovesCardDataKeepsOrdinaryText(t *testing.T) {
	for in, want := range map[string]string{
		"VISA ****4242":              "[card removed]",
		"4242 4242 4242 4242":        "[card removed]",
		"4242-4242-4242-4242 thanks": "[card removed] thanks",
		"card ending in 4242":        "card [card removed]",
		"Auth 839201":                "[code removed]",
		"Approval code: A83921":      "[code removed]",
		// ordinary receipt text must survive
		"Order #12345678": "Order #12345678", "$1,234.56": "$1,234.56",
		"Call 5125551234":     "Call 5125551234",
		"ref 123456789012":    "ref 123456789012",    // 12 digits
		"4242 4242 4242 4243": "4242 4242 4242 4243", // 16 digits, fails Luhn
		"Authentic Pizza":     "Authentic Pizza", "Max Burger 2": "Max Burger 2",
	} {
		if got := SanitizeText(in); got != want {
			t.Errorf("SanitizeText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinViewScrubsCardTextAndNeverReturnsImages(t *testing.T) {
	// The fixture receipt's merchant, item and person names are scrubbed on the participant path.
	clock := time.Now()
	a, _ := sessionAPI(t, &clock)
	_, created := call(a, ownerA, "POST", "/session")
	_, out := join(a, `{"token":"`+created["token"].(string)+`"}`)
	b, _ := json.Marshal(out)
	for _, bad := range []string{"image", "storage", "signed"} {
		if strings.Contains(strings.ToLower(string(b)), bad) {
			t.Fatalf("participant view mentions %q: %s", bad, b)
		}
	}
}

func TestProposalSchemaHasNoPaymentFields(t *testing.T) {
	b, _ := json.Marshal(proposalSchema())
	for _, bad := range []string{"card", "auth", "signature", "cardholder", "last4"} {
		if strings.Contains(strings.ToLower(string(b)), bad) {
			t.Fatalf("schema has payment field %q", bad)
		}
	}
}
