package print

import "testing"

func TestValidateRawAcceptsZPL(t *testing.T) {
	if err := ValidateRaw([]byte("^XA^FDhi^FS^XZ")); err != nil {
		t.Fatalf("expected ZPL to validate, got %v", err)
	}
}

func TestValidateRawAcceptsTSPL(t *testing.T) {
	if err := ValidateRaw([]byte("SIZE 89 mm, 36 mm\nCLS\nPRINT 1,1\n")); err != nil {
		t.Fatalf("expected TSPL to validate, got %v", err)
	}
}

func TestValidateRawAcceptsUnknownPassthrough(t *testing.T) {
	if err := ValidateRaw([]byte("\x1b@hello")); err != nil {
		t.Fatalf("expected unknown raw payload to pass through, got %v", err)
	}
}

func TestValidateRawRejectsEmpty(t *testing.T) {
	if err := ValidateRaw([]byte("")); err == nil {
		t.Fatal("expected empty raw to be rejected")
	}
}
