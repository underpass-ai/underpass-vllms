package twopass

import "testing"

func TestErrorReturnsMessage(t *testing.T) {
	err := &Error{Message: "boom"}
	if err.Error() != "boom" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
}

func TestIntermediateRepresentationSizeBytes(t *testing.T) {
	ir := IntermediateRepresentation("abc")
	if ir.SizeBytes() != 3 {
		t.Fatalf("unexpected size: %d", ir.SizeBytes())
	}
}
