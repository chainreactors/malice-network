package readline

import "testing"

func TestAIPredictionInsertText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		currentLine string
		prediction  string
		want        string
	}{
		{
			name:        "ArgumentBoundary_PreservesNoExtraSpace",
			currentLine: "build beacon --os ",
			prediction:  "windows",
			want:        "windows",
		},
		{
			name:        "ArgumentBoundary_TrimsLeadingWhitespace",
			currentLine: "build beacon --os ",
			prediction:  "   \twindows",
			want:        "windows",
		},
		{
			name:        "MidToken_CompletesCommandName",
			currentLine: "wiz",
			prediction:  "wizard",
			want:        "ard",
		},
		{
			name:        "MidToken_CompletesArgumentValue",
			currentLine: "build beacon --os w",
			prediction:  "windows",
			want:        "indows",
		},
		{
			name:        "MidToken_ExactMatchYieldsEmpty",
			currentLine: "wizard",
			prediction:  "wizard",
			want:        "",
		},
		{
			name:        "NextToken_WhenPredictionDoesNotMatchTokenPrefix",
			currentLine: "wizard",
			prediction:  "--help",
			want:        " --help",
		},
		{
			name:        "TabBoundary_TreatedAsWhitespace",
			currentLine: "connect\t",
			prediction:  "127.0.0.1",
			want:        "127.0.0.1",
		},
		{
			name:        "EmptyLine_NoLeadingSpace",
			currentLine: "",
			prediction:  "wizard",
			want:        "wizard",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := aiPredictionInsertText(tt.currentLine, tt.prediction); got != tt.want {
				t.Fatalf("aiPredictionInsertText(%q, %q) = %q, want %q", tt.currentLine, tt.prediction, got, tt.want)
			}
		})
	}
}
