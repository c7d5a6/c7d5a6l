package liquipedia_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
)

func TestLocalPlayerURL(t *testing.T) {
	got := liquipedia.LocalPlayerURL("starcraft", "Ever)P(NaBi")
	want := "local://starcraft/player/Ever)P(NaBi"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if !liquipedia.IsLocalPlayerURL(got) {
		t.Fatal("expected local")
	}
	norm, err := liquipedia.NormalizePlayerLink(got)
	if err != nil {
		t.Fatal(err)
	}
	if norm != want {
		t.Fatalf("normalize=%q want %q", norm, want)
	}
}
