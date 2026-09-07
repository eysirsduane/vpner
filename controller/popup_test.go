package controller

import (
	"reflect"
	"testing"
)

func TestPopupImageResponseValue(t *testing.T) {
	want := []string{
		"https://example.com/one.png",
		"https://example.com/two.png",
	}

	for _, input := range []string{
		`["https://example.com/one.png", "https://example.com/two.png"]`,
		"https://example.com/one.png, https://example.com/two.png",
	} {
		if got := popupImageResponseValue(input); !reflect.DeepEqual(got, want) {
			t.Fatalf("popupImageResponseValue(%q) = %#v, want %#v", input, got, want)
		}
	}

	if got := popupImageResponseValue("https://example.com/one.png"); !reflect.DeepEqual(got, want[:1]) {
		t.Fatalf("popupImageResponseValue() = %#v, want %#v", got, want[:1])
	}
}
