package model

import (
	"strings"
	"testing"
)

func TestGenerateRandomUserID(t *testing.T) {
	for i := 0; i < 100; i++ {
		id, err := GenerateRandomUserID()
		if err != nil {
			t.Fatalf("GenerateRandomUserID() error = %v", err)
		}
		if id < randomUserIDMin || id > randomUserIDMax {
			t.Fatalf("GenerateRandomUserID() = %d, want 9-digit user id", id)
		}
	}
}

func TestTransferCodeCharsExcludeAmbiguousCharacters(t *testing.T) {
	if strings.ContainsAny(transferCodeChars, "0ilo") {
		t.Fatalf("transferCodeChars contains ambiguous characters: %q", transferCodeChars)
	}
}

func TestGenerateRandomTransferCode(t *testing.T) {
	code, err := GenerateRandomTransferCode("app")
	if err != nil {
		t.Fatalf("GenerateRandomTransferCode() error = %v", err)
	}
	if !strings.HasPrefix(code, "app") {
		t.Fatalf("GenerateRandomTransferCode() = %q, want product prefix", code)
	}
	if len(code) != len("app")+transferCodeLength {
		t.Fatalf("GenerateRandomTransferCode() length = %d, want %d", len(code), len("app")+transferCodeLength)
	}
	for _, ch := range strings.TrimPrefix(code, "app") {
		if !strings.ContainsRune(transferCodeChars, ch) {
			t.Fatalf("GenerateRandomTransferCode() suffix contains invalid char %q", ch)
		}
	}
}
