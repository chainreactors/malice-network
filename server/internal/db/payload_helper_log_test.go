package db

import "testing"

func TestTailBuilderLog(t *testing.T) {
	tests := []struct {
		name  string
		log   string
		limit int
		want  string
	}{
		{
			name:  "trailing newline does not consume the line limit",
			log:   "line1\nline2\nline3\nline4\n",
			limit: 2,
			want:  "line3\nline4\n",
		},
		{
			name:  "log without trailing newline",
			log:   "line1\nline2\nline3",
			limit: 2,
			want:  "line2\nline3",
		},
		{
			name:  "CRLF line endings",
			log:   "line1\r\nline2\r\nline3\r\n",
			limit: 2,
			want:  "line2\r\nline3\r\n",
		},
		{
			name:  "blank lines count as log lines",
			log:   "line1\n\nline3\n",
			limit: 2,
			want:  "\nline3\n",
		},
		{
			name:  "zero returns the complete log",
			log:   "line1\nline2\n",
			limit: 0,
			want:  "line1\nline2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tailBuilderLog(tt.log, tt.limit); got != tt.want {
				t.Fatalf("tailBuilderLog() = %q, want %q", got, tt.want)
			}
		})
	}
}
