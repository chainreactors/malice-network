package core

import (
	"reflect"
	"testing"

	"github.com/reeflective/readline/internal/term"
)

// getTermWidth is used as a variable so that we can
// use specific terminal widths in our tests.
var getTermWidth = term.GetWidth

func TestLine_Insert(t *testing.T) {
	line := Line("multiple-ambiguous 10.203.23.45")

	type args struct {
		pos int
		r   []rune
	}
	tests := []struct {
		name string
		l    *Line
		args args
		want string
	}{
		{
			name: "Append to empty line",
			l:    new(Line),
			args: args{pos: 0, r: []rune("127.0.0.1")},
			want: "127.0.0.1",
		},
		{
			name: "Append to end of line",
			l:    &line,
			args: args{pos: len(line), r: []rune(" 127.0.0.1")},
			want: "multiple-ambiguous 10.203.23.45 127.0.0.1",
		},
		{
			name: "Insert at beginning of line",
			l:    &line,
			args: args{pos: 0, r: []rune("root ")},
			want: "root multiple-ambiguous 10.203.23.45 127.0.0.1",
		},
		{
			name: "Insert with an out of range position",
			l:    &line,
			args: args{pos: 100, r: []rune("dropped")},
			want: "root multiple-ambiguous 10.203.23.45 127.0.0.1",
		},
		{
			name: "Insert a null-terminated slice",
			l:    &line,
			args: args{pos: 0, r: []rune("example " + string(rune(0)))},
			want: "example root multiple-ambiguous 10.203.23.45 127.0.0.1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.l.Insert(test.args.pos, test.args.r...)
		})

		if string(*test.l) != test.want {
			t.Errorf("Line: '%s', wanted '%s'", string(*test.l), test.want)
		}
	}
}

func TestLine_InsertBetween(t *testing.T) {
	line := Line("multiple-ambiguous 10.203.23.45")

	type args struct {
		bpos int
		epos int
		r    []rune
	}
	tests := []struct {
		name string
		l    *Line
		args args
		want string
	}{
		{
			name: "Insert at beginning of line",
			l:    &line,
			args: args{bpos: 0, r: []rune("root ")},
			want: "root multiple-ambiguous 10.203.23.45",
		},
		{
			name: "Insert with a non-ending range (thus at other position)",
			l:    &line,
			args: args{bpos: 24, epos: -1, r: []rune("trail ")},
			want: "root multiple-ambiguous trail 10.203.23.45",
		},
		{
			name: "Append to end of line (no epos)",
			l:    &line,
			args: args{bpos: 42, epos: -1, r: []rune(" 127.0.0.1")},
			want: "root multiple-ambiguous trail 10.203.23.45 127.0.0.1",
		},

		{
			name: "Insert with cut",
			l:    &line,
			args: args{bpos: 23, epos: 29, r: []rune(" 10.10.10.10")},
			want: "root multiple-ambiguous 10.10.10.10 10.203.23.45 127.0.0.1",
		},
		{
			name: "Insert at invalid position (negative bpos/epos)",
			l:    &line,
			args: args{bpos: -1, epos: -1, r: []rune("root ")},
			want: "root multiple-ambiguous 10.10.10.10 10.203.23.45 127.0.0.1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.l.InsertBetween(test.args.bpos, test.args.epos, test.args.r...)
		})

		if string(*test.l) != test.want {
			t.Errorf("Line: '%s', wanted '%s'", string(*test.l), test.want)
		}
	}
}

func TestLine_Cut(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr")

	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name string
		l    *Line
		args args
		want string
	}{
		{
			name: "Cut in the middle",
			l:    &line,
			args: args{bpos: 21, epos: 29},
			want: "basic -f \"commands.go\" -cp=/usr",
		},
		{
			name: "Cut at beginning of line",
			l:    &line,
			args: args{bpos: 0, epos: 9},
			want: "\"commands.go\" -cp=/usr",
		},
		{
			name: "Cut with range out of bounds (epos greater than line)",
			l:    &line,
			args: args{bpos: 13, epos: len(line) + 1},
			want: "\"commands.go\"",
		},
		{
			name: "Cut trailing range (epos -1)",
			l:    &line,
			args: args{bpos: 9, epos: -1},
			want: "\"commands",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.l.Cut(test.args.bpos, test.args.epos)
		})

		if string(*test.l) != test.want {
			t.Errorf("Line: '%s', wanted '%s'", string(*test.l), test.want)
		}
	}
}

func TestLine_CutRune(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr")

	type args struct {
		pos int
	}
	tests := []struct {
		name string
		l    *Line
		args args
		want string
	}{
		{
			name: "Cut rune in the middle",
			l:    &line,
			args: args{pos: 22},
			want: "basic -f \"commands.go,ine.go\" -cp=/usr",
		},
		{
			name: "Cut rune at end of line, append mode",
			l:    &line,
			args: args{pos: len(line) - 1},
			want: "basic -f \"commands.go,ine.go\" -cp=/us",
		},
		{
			name: "Cut rune at invalid position (not removed)",
			l:    &line,
			args: args{pos: len(line) + 1},
			want: "basic -f \"commands.go,ine.go\" -cp=/us",
		},
		{
			name: "Cut rune at beginning",
			l:    &line,
			args: args{pos: 0},
			want: "asic -f \"commands.go,ine.go\" -cp=/us",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.l.CutRune(test.args.pos)
		})

		if string(*test.l) != test.want {
			t.Errorf("Line: '%s', wanted '%s'", string(*test.l), test.want)
		}
	}
}

func TestLine_Len(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr")

	tests := []struct {
		name string
		l    *Line
		want int
	}{
		{
			name: "Length of non-empty line",
			l:    &line,
			want: len(line),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.Len(); got != tt.want {
				t.Errorf("Line.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLine_SelectWord(t *testing.T) {
	line := Line("basic -c true -p on")

	type args struct {
		pos int
	}
	tests := []struct {
		name     string
		l        *Line
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Select word from start",
			l:        &line,
			args:     args{0},
			wantBpos: 0,
			wantEpos: 4,
		},
		{
			name:     "Select word in middle of word",
			l:        &line,
			args:     args{2},
			wantBpos: 0,
			wantEpos: 4,
		},
		{
			name:     "Select command flag",
			l:        &line,
			args:     args{10},
			wantBpos: 9,
			wantEpos: 12,
		},
		{
			name:     "Select numeric expression",
			l:        &line,
			args:     args{len(line)},
			wantBpos: len(line) - 2,
			wantEpos: len(line) - 1,
		},
		{
			name:     "Select between words (only select spaces)",
			l:        &line,
			args:     args{5},
			wantBpos: 5,
			wantEpos: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBpos, gotEpos := test.l.SelectWord(test.args.pos)
			if gotBpos != test.wantBpos {
				t.Errorf("Line.SelectWord() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Line.SelectWord() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestLine_SelectBlankWord(t *testing.T) {
	line := Line("basic -c true -f long-argument with 'quotes here'")

	type args struct {
		pos int
	}
	tests := []struct {
		name     string
		l        *Line
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Select word from start",
			l:        &line,
			args:     args{0},
			wantBpos: 0,
			wantEpos: 4,
		},
		{
			name:     "Select word in middle of word",
			l:        &line,
			args:     args{2},
			wantBpos: 0,
			wantEpos: 4,
		},
		{
			name:     "Select command flag",
			l:        &line,
			args:     args{10},
			wantBpos: 9,
			wantEpos: 12,
		},
		{
			name:     "Select dash word",
			l:        &line,
			args:     args{21},
			wantBpos: 17,
			wantEpos: 29,
		},
		{
			name:     "Select in middle of shell word (quoted)",
			l:        &line,
			args:     args{len(line)},
			wantBpos: 44,
			wantEpos: 48,
		},
		{
			name:     "Select between words (only select spaces)",
			l:        &line,
			args:     args{30},
			wantBpos: 30,
			wantEpos: 30,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBpos, gotEpos := test.l.SelectBlankWord(test.args.pos)
			if gotBpos != test.wantBpos {
				t.Errorf("Line.SelectWord() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Line.SelectWord() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestLine_Find(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")
	pos := 0 // We reuse the same updated position for each next test case.

	type args struct {
		r       rune
		pos     int
		forward bool
	}
	tests := []struct {
		name     string
		l        *Line
		args     args
		wantRpos int
	}{
		// Forward
		{
			name:     "Find on empty line",
			l:        new(Line),
			args:     args{r: '"', pos: pos, forward: true},
			wantRpos: -1,
		},
		{
			name:     "Find first quote from beginning of line",
			l:        &line,
			args:     args{r: '"', pos: pos, forward: true},
			wantRpos: 9,
		},
		{
			name:     "Find first opening bracket from start",
			l:        &line,
			args:     args{r: '[', pos: pos, forward: true},
			wantRpos: 49,
		},
		{
			name:     "Search for non existent rune in the line",
			l:        &line,
			args:     args{r: '%', pos: pos, forward: true},
			wantRpos: -1,
		},
		// Backward
		{
			name:     "Find first quote from end of line",
			l:        &line,
			args:     args{r: '"', pos: len(line), forward: false},
			wantRpos: 29,
		},
		{
			name:     "Find first opening bracket from end of line",
			l:        &line,
			args:     args{r: '[', pos: len(line), forward: false},
			wantRpos: 49,
		},
		{
			name:     "Find first opening bracket from end of line (out-of-range)",
			l:        &line,
			args:     args{r: '[', pos: len(line) + 1, forward: false},
			wantRpos: 49,
		},
		{
			name:     "Search for non existent rune in the line",
			l:        &line,
			args:     args{r: '%', pos: len(line), forward: false},
			wantRpos: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRpos := tt.l.Find(tt.args.r, tt.args.pos, tt.args.forward); gotRpos != tt.wantRpos {
				t.Errorf("Line.Next() = %v, want %v", gotRpos, tt.wantRpos)
			}

			pos = tt.wantRpos
		})
	}
}

func TestLine_FindSurround(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")

	type args struct {
		r   rune
		pos int
	}
	tests := []struct {
		name      string
		l         *Line
		args      args
		wantBpos  int
		wantEpos  int
		wantBchar rune
		wantEchar rune
	}{
		{
			name:      "Find double quotes (success)",
			l:         &line,
			args:      args{r: '"', pos: 15},
			wantBpos:  9,
			wantEpos:  29,
			wantBchar: '"',
			wantEchar: '"',
		},
		{
			name:      "Find double quotes (fail)",
			l:         &line,
			args:      args{r: '"', pos: 0},
			wantBpos:  -1,
			wantEpos:  9,
			wantBchar: '"',
			wantEchar: '"',
		},
		{
			name:      "Find brackets (success)",
			l:         &line,
			args:      args{r: '[', pos: line.Len() - 1},
			wantBpos:  line.Len() - 15,
			wantEpos:  line.Len() - 1,
			wantBchar: '[',
			wantEchar: ']',
		},
		{
			name:      "Find brackets (fail)",
			l:         &line,
			args:      args{r: '[', pos: 35},
			wantBpos:  -1,
			wantEpos:  line.Len() - 1,
			wantBchar: '[',
			wantEchar: ']',
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBpos, gotEpos, gotBchar, gotEchar := test.l.FindSurround(test.args.r, test.args.pos)

			// Pos
			if gotBpos != test.wantBpos {
				t.Errorf("Line.FindSurround() (bpos) = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Line.FindSurround() (epos) = %v, want %v", gotEpos, test.wantEpos)
			}

			// Chars
			if gotBchar != test.wantBchar {
				t.Errorf("Line.FindSurround() (bchar) = %v, want %v", gotBchar, test.wantBchar)
			}

			if gotEchar != test.wantEchar {
				t.Errorf("Line.FindSurround() (gotEchar) = %v, want %v", gotEchar, test.wantEchar)
			}
		})
	}
}

func TestLine_SurroundQuotes(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr \"another\" --option 'value1 value2'")

	type args struct {
		single bool
		pos    int
	}
	tests := []struct {
		name     string
		l        *Line
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Find double quotes (success)",
			l:        &line,
			args:     args{single: false, pos: 15},
			wantBpos: 9,
			wantEpos: 29,
		},
		{
			name:     "Find double quotes (fail)",
			l:        &line,
			args:     args{single: false, pos: 0},
			wantBpos: -1,
			wantEpos: 9,
		},
		{
			name:     "Find double quotes (fail not surrounding pos)",
			l:        &line,
			args:     args{single: false, pos: 35},
			wantBpos: -1,
			wantEpos: -1,
		},
		{
			name:     "Find single quotes (success)",
			l:        &line,
			args:     args{single: true, pos: line.Len() - 3},
			wantBpos: line.Len() - 15,
			wantEpos: line.Len() - 1,
		},
		{
			name:     "Find single quotes (fail)",
			l:        &line,
			args:     args{single: true, pos: 35},
			wantBpos: -1,
			wantEpos: line.Len() - 15,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBpos, gotEpos := test.l.SurroundQuotes(test.args.single, test.args.pos)

			// Pos
			if gotBpos != test.wantBpos {
				t.Errorf("Line.SurroundQuotes() (bpos) = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Line.SurroundQuotes() (epos) = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestLine_Lines(t *testing.T) {
	tests := []struct {
		name      string
		l         Line
		wantLines int
	}{
		{
			name:      "Single line",
			l:         Line("lonely\twords end\\n here"),
			wantLines: 0,
		},
		{
			name:      "Empty line (middle)",
			l:         Line("lonely\n\twords\n\nend\\n here"),
			wantLines: 3,
		},
		{
			name:      "Empty line (trailing & middle)",
			l:         Line("lonely\n\twords\n\nend\\n here\n"),
			wantLines: 4,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotLines := test.l.Lines()

			if gotLines != test.wantLines {
				t.Errorf("Line.Lines() = %v, want %v", gotLines, test.wantLines)
			}
		})
	}
}

func TestLine_Forward(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")

	type args struct {
		split Tokenizer
		pos   int
	}
	tests := []struct {
		name       string
		l          *Line
		args       args
		wantAdjust int
	}{
		{
			name:       "Forward word",
			l:          &line,
			args:       args{split: line.Tokenize, pos: 0},
			wantAdjust: 6,
		},
		{
			name:       "Forward blank word",
			l:          &line,
			args:       args{split: line.TokenizeSpace, pos: 10},
			wantAdjust: 21,
		},
		{
			name:       "Forward bracket",
			l:          &line,
			args:       args{split: line.TokenizeBlock, pos: 49},
			wantAdjust: 64,
		},

		{
			name:       "Forward bracket (no match)",
			l:          &line,
			args:       args{split: line.TokenizeBlock, pos: 48},
			wantAdjust: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if gotAdjust := test.l.Forward(test.args.split, test.args.pos); gotAdjust != test.wantAdjust {
				t.Errorf("Line.Forward() = %v, want %v", gotAdjust, test.wantAdjust)
			}
		})
	}
}

func TestLine_ForwardEnd(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")
	emptyLine := new(Line)

	type args struct {
		split Tokenizer
		pos   int
	}
	tests := []struct {
		name       string
		l          *Line
		args       args
		wantAdjust int
	}{
		{
			name:       "Forward word end (empty line)",
			l:          emptyLine,
			args:       args{split: emptyLine.Tokenize, pos: 0},
			wantAdjust: 0,
		},
		{
			name:       "Forward word end (beginning of line)",
			l:          &line,
			args:       args{split: line.Tokenize, pos: 0},
			wantAdjust: 4,
		},
		{
			name:       "Forward word end (end of line)",
			l:          &line,
			args:       args{split: line.Tokenize, pos: line.Len()},
			wantAdjust: 0,
		},
		{
			name:       "Forward blank word end (middle of line)",
			l:          &line,
			args:       args{split: line.TokenizeSpace, pos: 10},
			wantAdjust: 19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotAdjust := tt.l.ForwardEnd(tt.args.split, tt.args.pos); gotAdjust != tt.wantAdjust {
				t.Errorf("Line.ForwardEnd() = %v, want %v", gotAdjust, tt.wantAdjust)
			}
		})
	}
}

func TestLine_Backward(t *testing.T) {
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")
	emptyLine := new(Line)

	type args struct {
		split Tokenizer
		pos   int
	}
	tests := []struct {
		name       string
		l          *Line
		args       args
		wantAdjust int
	}{
		{
			name:       "Backward word (empty line)",
			l:          emptyLine,
			args:       args{split: emptyLine.Tokenize, pos: 0},
			wantAdjust: 0,
		},
		{
			name:       "Backward word (beginning of line)",
			l:          &line,
			args:       args{split: line.Tokenize, pos: 0},
			wantAdjust: 0,
		},
		{
			name:       "Backward word (middle of line)",
			l:          &line,
			args:       args{split: line.Tokenize, pos: 6},
			wantAdjust: -6,
		},
		{
			name:       "Backward blank word",
			l:          &line,
			args:       args{split: line.TokenizeSpace, pos: 22},
			wantAdjust: -13,
		},
		{
			name:       "Backward bracket",
			l:          &line,
			args:       args{split: line.TokenizeBlock, pos: line.Len() - 1},
			wantAdjust: -14,
		},

		{
			name:       "Backward bracket (no match)",
			l:          &line,
			args:       args{split: line.TokenizeBlock, pos: line.Len() - 2},
			wantAdjust: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotAdjust := tt.l.Backward(tt.args.split, tt.args.pos); gotAdjust != tt.wantAdjust {
				t.Errorf("Line.Backward() = %v, want %v", gotAdjust, tt.wantAdjust)
			}
		})
	}
}

func TestLine_Tokenize(t *testing.T) {
	line := Line("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")
	emptyLines := Line("basic -f \"commands.go \n\nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		pos int
	}
	tests := []struct {
		name  string
		l     *Line
		args  args
		want  []string
		want1 int
		want2 int
	}{
		{
			name: "Tokenize line",
			args: args{pos: line.Len() - 1},
			l:    &line,
			want: []string{
				"basic ", "-", "f ", "\"", "commands", ".", "go \n",
				"another ", "testing", "\" ", "--", "alternate ", "\"", "another\n",
				"quote", "\" ", "-", "c",
			},
			want1: 17,
			want2: 0,
		},
		{
			name: "Tokenize line (with empty lines)",
			args: args{pos: line.Len() - 1},
			l:    &emptyLines,
			want: []string{
				"basic ", "-", "f ", "\"", "commands", ".", "go \n", "\n",
				"another ", "testing", "\" ", "--", "alternate ", "\"", "another\n",
				"quote", "\" ", "-", "c",
			},
			want1: 17,
			want2: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, got1, got2 := test.l.Tokenize(test.args.pos)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Line.Tokenize() got = %v, want %v", got, test.want)
			}

			if got1 != test.want1 {
				t.Errorf("Line.Tokenize() got1 = %v, want %v", got1, test.want1)
			}

			if got2 != test.want2 {
				t.Errorf("Line.Tokenize() got2 = %v, want %v", got2, test.want2)
			}
		})
	}
}

func TestLine_TokenizeSpace(t *testing.T) {
	line := Line("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")
	emptyLine := new(Line)
	emptyLines := Line("basic -f \"commands.go \n\nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		pos int
	}
	tests := []struct {
		name  string
		l     *Line
		args  args
		want  []string
		want1 int
		want2 int
	}{
		{
			name:  "Empty line",
			args:  args{0},
			l:     emptyLine,
			want1: 0,
			want2: 0,
		},
		{
			name: "With newlines (cursor append)",
			args: args{pos: line.Len() - 1},
			l:    &line,
			want: []string{
				"basic ", "-f ", "\"commands.go \n", "another ", "testing\" ", "--alternate ", "\"another\n", "quote\" ", "-c",
			},
			want1: 8, // equal len(want) -1 since we are at the end of the line.
			want2: 1,
		},
		{
			name: "With empty lines (cursor on last char)",
			args: args{pos: emptyLines.Len()},
			l:    &emptyLines,
			want: []string{
				"basic ", "-f ", "\"commands.go \n", "\n", "another ", "testing\" ", "--alternate ", "\"another\n", "quote\" ", "-c",
			},
			want1: 9, // equal len(want) -1 since we are at the end of the line.
			want2: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, got1, got2 := test.l.TokenizeSpace(test.args.pos)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Line.TokenizeSpace() got = %v, want %v", got, test.want)
			}

			if got1 != test.want1 {
				t.Errorf("Line.TokenizeSpace() got1 = %v, want %v", got1, test.want1)
			}

			if got2 != test.want2 {
				t.Errorf("Line.TokenizeSpace() got2 = %v, want %v", got2, test.want2)
			}
		})
	}
}

func TestLine_TokenizeBlock(t *testing.T) {
	noBlocks := Line("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\"")
	blockStart := Line("{ expression here } -a [value1 value2]")
	line := Line("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -v { expression here } -a [value1 value2]")
	quotedLine := Line("basic -f \"commands.go \nanother testing\" '--alternate \"another\nquote\" -v { expression here }' -a [value1 value2]")
	emptyLine := new(Line)

	type args struct {
		pos int
	}
	tests := []struct {
		name  string
		l     *Line
		args  args
		want  []string
		want1 int
		want2 int
	}{
		{
			name:  "Empty line",
			args:  args{0},
			l:     emptyLine,
			want1: 0,
			want2: 0,
		},
		{
			name:  "No blocks",
			args:  args{noBlocks.Len()},
			l:     &noBlocks,
			want1: 0,
			want2: 0,
		},
		{
			name:  "Braces at line start, cursor on closing brace (find/move)",
			args:  args{18},
			l:     &blockStart,
			want:  []string{"", "{ expression here "},
			want1: 1,
			want2: 18,
		},
		{
			name:  "Braces at line start, cursor at line start (find/move)",
			args:  args{0},
			l:     &blockStart,
			want:  []string{"", "{ expression here "},
			want1: 1,
			want2: 0,
		},
		{
			name: "With newlines (cursor append) (find/move)",
			args: args{pos: line.Len()},
			l:    &line,
			want: []string{
				"basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -v { expression here } -a", "[value1 value2",
			},
			want1: 1,
			want2: 14,
		},
		{
			name: "With newlines (cursor on closing brace) (find/move)",
			args: args{pos: 89}, // 71
			l:    &line,
			want: []string{
				"basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -v", "{ expression here ",
			},
			want1: 1,
			want2: 18,
		},
		{
			name: "With newlines (cursor on open brace) (fail)",
			args: args{pos: 71},
			l:    &line,
			want: []string{
				"basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -v", "{ expression here ",
			},
			want1: 1,
			want2: 0,
		},
		{
			name:  "With braces inside quotes (cursor on closing brace) (fail)",
			args:  args{pos: 90},
			l:     &quotedLine,
			want:  nil,
			want1: 0,
			want2: 0,
		},
		{
			name:  "With braces inside quotes (cursor on opening brace) (fail)",
			args:  args{pos: 72},
			l:     &quotedLine,
			want:  nil,
			want1: 0,
			want2: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, got1, got2 := test.l.TokenizeBlock(test.args.pos)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Line.TokenizeBlock() got = %v, want %v", got, test.want)
			}

			if got1 != test.want1 {
				t.Errorf("Line.TokenizeBlock() got1 = %v, want %v", got1, test.want1)
			}

			if got2 != test.want2 {
				t.Errorf("Line.TokenizeBlock() got2 = %v, want %v", got2, test.want2)
			}
		})
	}
}

func TestDisplayLine(t *testing.T) {
	type args struct {
		indent int
	}
	tests := []struct {
		name string
		l    *Line
		args args
	}{
		// TODO: Add test cases.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DisplayLine(tt.l, tt.args.indent)
		})
	}
}

func TestCoordinatesLine(t *testing.T) {
	indent := 10
	line := Line("basic -f \"commands.go,line.go\" -cp=/usr --option [value1 value2]")
	multiline := Line("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -v { expression here } -a [value1 value2]")

	// Reassign the function for getting the terminal width to a fixed value
	getTermWidth = func() int { return 80 }

	type args struct {
		indent    int
		suggested string
	}
	tests := []struct {
		name  string
		l     *Line
		args  args
		wantX int
		wantY int
	}{
		{
			name:  "Single line buffer",
			l:     &line,
			args:  args{indent: indent},
			wantY: 0,
			wantX: indent + 64,
		},
		{
			name:  "Multiline buffer",
			l:     &multiline,
			args:  args{indent: indent},
			wantY: 2,
			wantX: indent + 48,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotX, gotY := CoordinatesLine(test.l, test.args.indent)
			if gotX != test.wantX {
				t.Errorf("CoordinatesLine() gotX = %v, want %v", gotX, test.wantX)
			}

			if gotY != test.wantY {
				t.Errorf("CoordinatesLine() gotY = %v, want %v", gotY, test.wantY)
			}
		})
	}
}
