package gemini

import "testing"

func TestParseDataURL(t *testing.T) {
	mime, data, err := parseDataURL("data:image/png;base64,aGVsbG8=")
	if err != nil { t.Fatalf("parseDataURL() error = %v", err) }
	if mime != "image/png" || data != "aGVsbG8=" { t.Fatalf("unexpected result: %s %s", mime, data) }
}

func TestParseDataURLRejectsNonBase64(t *testing.T) {
	if _, _, err := parseDataURL("data:image/png,hello"); err == nil { t.Fatal("expected error") }
}
