package relay

import "testing"

func TestRedactImageURLRemovesCredentialsAndTracking(t *testing.T) {
	got := redactImageURL("https://user:secret@example.com/image.png?token=private#fragment")
	if got != "https://example.com/image.png" {
		t.Fatalf("redacted URL = %q", got)
	}
	if got := redactImageURL("data:image/png;base64,secret"); got != "[redacted]" {
		t.Fatalf("data URL redaction = %q", got)
	}
}
