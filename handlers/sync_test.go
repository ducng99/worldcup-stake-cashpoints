package handlers

import "testing"

func TestNextMatchStatus(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		incoming string
		want     string
	}{
		{name: "new match stores incoming as app status", incoming: "TIMED", want: "UPCOMING"},
		{name: "scheduled can become timed", previous: "SCHEDULED", incoming: "TIMED", want: "UPCOMING"},
		{name: "timed can become live", previous: "TIMED", incoming: "IN_PLAY", want: "LIVE"},
		{name: "live cannot become timed", previous: "IN_PLAY", incoming: "TIMED", want: "LIVE"},
		{name: "live can become finished", previous: "IN_PLAY", incoming: "FINISHED", want: "FINISHED"},
		{name: "finished cannot become live", previous: "FINISHED", incoming: "IN_PLAY", want: "FINISHED"},
		{name: "finished cannot become timed", previous: "FINISHED", incoming: "TIMED", want: "FINISHED"},
		{name: "cancelled stores as finished", previous: "UPCOMING", incoming: "CANCELLED", want: "FINISHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextMatchStatus(tt.previous, tt.incoming); got != tt.want {
				t.Fatalf("nextMatchStatus(%q, %q) = %q, want %q", tt.previous, tt.incoming, got, tt.want)
			}
		})
	}
}
