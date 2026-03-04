package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{input: "30d", want: 30 * 24 * time.Hour},
		{input: "1d", want: 24 * time.Hour},
		{input: "720h", want: 720 * time.Hour},
		{input: "24h30m", want: 24*time.Hour + 30*time.Minute},
		{input: "0d", wantErr: true},
		{input: "-1d", wantErr: true},
		{input: "abcd", wantErr: true},
		{input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
