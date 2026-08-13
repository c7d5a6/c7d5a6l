package liquipedia_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
)

func TestValidateAssetURL(t *testing.T) {
	got, err := liquipedia.ValidateAssetURL("https://liquipedia.net/commons/images/6/60/JD.png")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://liquipedia.net/commons/images/6/60/JD.png"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	if _, err := liquipedia.ValidateAssetURL("https://example.com/x.png"); err == nil {
		t.Fatal("expected host error")
	}
}
