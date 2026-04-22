package twopassapp

import "testing"

func TestSanitizeIntermediateExtractsFinalIRFence(t *testing.T) {
	raw := "**Reasoning:** brief note\n\n**Final Intermediate Representation:**\n```json\n{\"kind\":\"bug\",\"severity\":\"High\"}\n```"

	got := sanitizeIntermediate(raw)

	want := "{\"kind\":\"bug\",\"severity\":\"High\"}"
	if got != want {
		t.Fatalf("expected sanitized intermediate %q, got %q", want, got)
	}
}

func TestSanitizeIntermediateLeavesPlainIRUntouched(t *testing.T) {
	raw := "kind=bug\nseverity=High"

	got := sanitizeIntermediate(raw)

	if got != raw {
		t.Fatalf("expected plain IR to remain unchanged, got %q", got)
	}
}
