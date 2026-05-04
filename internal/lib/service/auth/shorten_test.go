package auth

import "testing"

func TestShortenName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"three words", "Иванов Иван Иванович", "И. Иванов"},
		{"single word unchanged", "Иванов", "Иванов"},
		{"empty string unchanged", "", ""},
		{"two-space separator", "Иванов  Иван", "И. Иванов"},
		{"latin three words", "Smith John Edward", "J. Smith"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShortenName(tc.in)
			if got != tc.want {
				t.Errorf("ShortenName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
