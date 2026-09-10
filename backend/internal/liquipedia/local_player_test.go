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
	if got := liquipedia.LocalPlayerName(want); got != "Ever)P(NaBi" {
		t.Fatalf("LocalPlayerName=%q", got)
	}
	hangul := liquipedia.LocalPlayerURL("starcraft", "홍광민")
	if got := liquipedia.LocalPlayerName(hangul); got != "홍광민" {
		t.Fatalf("LocalPlayerName hangul=%q from %q", got, hangul)
	}
}
