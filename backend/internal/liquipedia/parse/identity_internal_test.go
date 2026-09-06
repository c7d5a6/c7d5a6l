package parse

import "testing"

func TestSplitLatinIDKoreanRealName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		id       string
		realName string
		ok       bool
	}{
		{"LastHerO 김홍철", "LastHerO", "김홍철", true},
		{"Rush", "", "", false},
		{"오조은GOOD", "", "", false},
		{"Bisu", "", "", false},
		{"  SnOw  이영호 ", "SnOw", "이영호", true},
	}
	for _, tc := range cases {
		id, real, ok := splitLatinIDKoreanRealName(tc.in)
		if ok != tc.ok || id != tc.id || real != tc.realName {
			t.Fatalf("split(%q)=%q,%q,%v want %q,%q,%v", tc.in, id, real, ok, tc.id, tc.realName, tc.ok)
		}
	}
}
