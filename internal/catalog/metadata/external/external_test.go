package external

import "testing"

func TestLuceneQuote(t *testing.T) {
	cases := map[string]string{
		`Abbey Road`:        `"Abbey Road"`,
		`AC/DC`:             `"AC/DC"`,
		`Say "Hello"`:       `"Say \"Hello\""`,
		`back\slash`:        `"back\\slash"`,
	}
	for in, want := range cases {
		if got := luceneQuote(in); got != want {
			t.Errorf("luceneQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
