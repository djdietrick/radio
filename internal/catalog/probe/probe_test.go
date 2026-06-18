package probe

import "testing"

func TestParseDurationMs(t *testing.T) {
	cases := []struct {
		name string
		json string
		want int64
		ok   bool
	}{
		{"normal", `{"format":{"duration":"243.21"}}`, 243210, true},
		{"integer seconds", `{"format":{"duration":"60"}}`, 60000, true},
		{"missing duration", `{"format":{}}`, 0, true},
		{"empty object", `{}`, 0, true},
		{"negative clamps to zero", `{"format":{"duration":"-5"}}`, 0, true},
		{"non-numeric duration", `{"format":{"duration":"N/A"}}`, 0, false},
		{"malformed json", `{not json`, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseDurationMs([]byte(c.json))
			if c.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !c.ok && err == nil {
				t.Fatalf("expected an error, got none")
			}
			if got != c.want {
				t.Fatalf("got %d, want %d", got, c.want)
			}
		})
	}
}
