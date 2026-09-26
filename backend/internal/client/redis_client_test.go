package client

import (
	"testing"
	"time"
)

func TestDurationUntilUTCMidnight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		at   time.Time
		want time.Duration
	}{
		{
			name: "start of day",
			at:   time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC),
			want: 24 * time.Hour,
		},
		{
			name: "noon",
			at:   time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
			want: 12 * time.Hour,
		},
		{
			name: "one second before midnight",
			at:   time.Date(2026, 9, 26, 23, 59, 59, 0, time.UTC),
			want: time.Second,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := durationUntilUTCMidnight(tt.at)
			if got != tt.want {
				t.Fatalf("durationUntilUTCMidnight(%s) = %s, want %s", tt.at, got, tt.want)
			}
		})
	}
}
