package handlers

import "testing"

func TestNextMatchStatus(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		incoming string
		want     string
	}{
		{name: "new match uses incoming", incoming: "TIMED", want: "TIMED"},
		{name: "scheduled can become timed", previous: "SCHEDULED", incoming: "TIMED", want: "TIMED"},
		{name: "timed can become live", previous: "TIMED", incoming: "IN_PLAY", want: "IN_PLAY"},
		{name: "live cannot become timed", previous: "IN_PLAY", incoming: "TIMED", want: "IN_PLAY"},
		{name: "live can become finished", previous: "IN_PLAY", incoming: "FINISHED", want: "FINISHED"},
		{name: "finished cannot become live", previous: "FINISHED", incoming: "IN_PLAY", want: "FINISHED"},
		{name: "finished cannot become timed", previous: "FINISHED", incoming: "TIMED", want: "FINISHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextMatchStatus(tt.previous, tt.incoming); got != tt.want {
				t.Fatalf("nextMatchStatus(%q, %q) = %q, want %q", tt.previous, tt.incoming, got, tt.want)
			}
		})
	}
}
