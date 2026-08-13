package liquipedia_test

import (
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
)

func TestValidateURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		wantOK  bool
		wantOut string
	}{
		{
			name:    "starcraft page",
			in:      "https://liquipedia.net/starcraft/ASL/20",
			wantOK:  true,
			wantOut: "https://liquipedia.net/starcraft/ASL/20",
		},
		{
			name:    "other game path",
			in:      "https://liquipedia.net/counterstrike/Majors",
			wantOK:  true,
			wantOut: "https://liquipedia.net/counterstrike/Majors",
		},
		{
			name:    "www host",
			in:      "https://www.liquipedia.net/starcraft/ASL/19",
			wantOK:  true,
			wantOut: "https://liquipedia.net/starcraft/ASL/19",
		},
		{
			name:    "player disambiguation parens",
			in:      "https://liquipedia.net/starcraft/Larva_(Player)",
			wantOK:  true,
			wantOut: "https://liquipedia.net/starcraft/Larva_(Player)",
		},
		{
			name:    "player disambiguation encoded parens",
			in:      "https://liquipedia.net/starcraft/Larva_%28Player%29",
			wantOK:  true,
			wantOut: "https://liquipedia.net/starcraft/Larva_(Player)",
		},
		{
			name:   "root only",
			in:     "https://liquipedia.net/",
			wantOK: false,
		},
		{
			name:   "wiki only",
			in:     "https://liquipedia.net/starcraft",
			wantOK: false,
		},
		{
			name:   "http",
			in:     "http://liquipedia.net/starcraft/ASL/20",
			wantOK: false,
		},
		{
			name:   "other host",
			in:     "https://example.com/starcraft/ASL/20",
			wantOK: false,
		},
		{
			name:   "empty",
			in:     "  ",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := liquipedia.ValidateURL(tc.in)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if out != tc.wantOut {
					t.Fatalf("got %q, want %q", out, tc.wantOut)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got %q", out)
			}
		})
	}
}

func TestParsePageRef(t *testing.T) {
	t.Parallel()

	ref, err := liquipedia.ParsePageRef("https://liquipedia.net/starcraft/AfreecaTV/StarCraft_League_Remastered/14")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Wiki != "starcraft" {
		t.Fatalf("wiki=%q", ref.Wiki)
	}
	if ref.Title != "AfreecaTV/StarCraft_League_Remastered/14" {
		t.Fatalf("title=%q", ref.Title)
	}
}
