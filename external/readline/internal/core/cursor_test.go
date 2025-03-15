package core

import (
	"strings"
	"testing"
)

var (
	// cursorLine is a simple command line to test basic things on the cursor.
	cursorLine = Line("git command -c BranchName --another-opt value")

	// cursorMultiline is used for tests requiring multiline input (horizontal positions, etc).
	cursorMultiline = Line("git command -c \n second line of input before an empty line \n\n and then a last one")
)

func TestNewCursor(t *testing.T) {
	line := Line("test line")
	cursor := NewCursor(&line)

	if cursor.pos != 0 {
		t.Errorf("Cursor position: %d, should be %d", cursor.pos, 0)
	}

	if cursor.mark != -1 {
		t.Errorf("Cursor mark: %d, should be %d", cursor.mark, -1)
	}

	if cursor.line != &line {
		t.Errorf("Cursor line: %d, should be %d", cursor.line, line)
	}
}

func TestCursor_Set(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	type args struct {
		pos int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected int
	}{
		{
			name:     "Valid position",
			args:     args{10},
			fields:   fields{line: &cursorLine},
			expected: 10,
		},
		{
			name:     "Bigger than line length",
			args:     args{100},
			fields:   fields{line: &cursorLine},
			expected: len(cursorLine),
		},
		{
			name:     "Negative",
			args:     args{-1},
			fields:   fields{line: &cursorLine, pos: 5},
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			c.Set(test.args.pos)

			if c.pos != test.expected {
				t.Errorf("Cursor position: %d, should be %d", c.pos, test.expected)
			}
		})
	}
}

func TestCursor_Pos(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Valid cursor position",
			fields: fields{line: &cursorLine, pos: 10},
			want:   10,
		},
		{
			name:   "Out-of-range cursor position",
			fields: fields{line: &cursorLine, pos: 100},
			want:   len(cursorLine),
		},
		{
			name:   "Negative cursor position",
			fields: fields{line: &cursorLine, pos: -1},
			want:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			if got := c.Pos(); got != test.want {
				t.Errorf("Cursor.Pos() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_Inc(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Start cursor position",
			fields: fields{line: &cursorLine, pos: 0},
			want:   1,
		},
		{
			name:   "Before end of line",
			fields: fields{line: &cursorLine, pos: len(cursorLine) - 1},
			want:   len(cursorLine),
		},
		{
			name:   "End of line",
			fields: fields{line: &cursorLine, pos: len(cursorLine)},
			want:   len(cursorLine),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			c.Inc()

			if got := c.Pos(); got != test.want {
				t.Errorf("Cursor.Pos() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_Dec(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Middle of line",
			fields: fields{line: &cursorLine, pos: 10},
			want:   9,
		},
		{
			name:   "Before beginning of line",
			fields: fields{line: &cursorLine, pos: 1},
			want:   0,
		},
		{
			name:   "Beginning of line",
			fields: fields{line: &cursorLine, pos: 0},
			want:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			c.Dec()

			if got := c.Pos(); got != test.want {
				t.Errorf("Cursor.Pos() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_Move(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	type args struct {
		offset int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Valid positive offset",
			args:   args{10},
			fields: fields{line: &cursorLine, pos: 5},
			want:   15,
		},
		{
			name:   "Valid negative offset",
			args:   args{-10},
			fields: fields{line: &cursorLine, pos: 15},
			want:   5,
		},
		{
			name:   "Out-of-bound positive offset",
			args:   args{10},
			fields: fields{line: &cursorLine, pos: len(cursorLine) - 5},
			want:   len(cursorLine),
		},
		{
			name:   "Out-of-bound negative offset",
			args:   args{-10},
			fields: fields{line: &cursorLine, pos: 5},
			want:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			c.Move(test.args.offset)

			if got := c.Pos(); got != test.want {
				t.Errorf("Cursor.Pos() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_Char(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   rune
	}{
		{
			name:   "Valid position",
			fields: fields{line: &cursorLine, pos: 5},
			want:   'o',
		},
		{
			name:   "Negative position (start char)",
			fields: fields{line: &cursorLine, pos: -1},
			want:   'g',
		},
		{
			name:   "Empty line",
			fields: fields{line: new(Line), pos: 0},
			want:   0,
		},
		{
			name:   "Append-mode position",
			fields: fields{line: &cursorLine, pos: cursorLine.Len()},
			want:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}

			if cur.Char() != test.want {
				t.Errorf("Cursor.Char() = %v, want %v", cur.Char(), test.want)
			}
		})
	}
}

func TestCursor_ReplaceWith(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	type args struct {
		char rune
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rune
	}{
		{
			name:   "Valid position",
			fields: fields{line: &cursorLine, pos: 0},
			args:   args{char: 's'},
			want:   's',
		},
		{
			name:   "Negative position (start char)",
			fields: fields{line: &cursorLine, pos: -1},
			args:   args{char: 's'},
			want:   's',
		},
		{
			name:   "Empty line (equivalent to append-mode)",
			fields: fields{line: new(Line), pos: 0},
			args:   args{char: 's'},
			want:   's',
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.ReplaceWith(test.args.char)

			if cur.Char() != test.want {
				t.Errorf("Cursor.Char() = %v, want %v", cur.Char(), test.want)
			}
		})
	}
}

func TestCursor_ToFirstNonSpace(t *testing.T) {
	tabLine := Line("\t git command")

	type fields struct {
		pos  int
		mark int
		line *Line
	}
	type args struct {
		forward bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Empty line",
			fields: fields{line: new(Line)},
			args:   args{forward: true},
			want:   0,
		},
		{
			name:   "Single line (on space)",
			fields: fields{line: &cursorLine, pos: 3},
			args:   args{forward: true},
			want:   4,
		},
		{
			name:   "Single line (tab beginning)",
			fields: fields{line: &tabLine, pos: 1},
			args:   args{forward: false},
			want:   0,
		},
		{
			name:   "Single line (beginning of line)",
			fields: fields{line: &cursorLine, pos: 0},
			args:   args{forward: false},
			want:   0,
		},
		{
			name:   "Single line backward (on space)",
			fields: fields{line: &cursorLine, pos: 3},
			args:   args{forward: false},
			want:   2,
		},
		{
			name:   "Single line forward (end-of-line backward)",
			fields: fields{line: &cursorLine, pos: cursorLine.Len()},
			args:   args{forward: true},
			want:   cursorLine.Len() - 1,
		},
		{
			name:   "Multiline line forward",
			fields: fields{line: &cursorMultiline, pos: 14},
			args:   args{forward: true},
			want:   17,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.ToFirstNonSpace(test.args.forward)

			if got := cur.Pos(); got != test.want {
				t.Errorf("Cursor.ToFirstNonSpace() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_BeginningOfLine(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Single line",
			fields: fields{line: &cursorLine, pos: 5},
			want:   0,
		},
		{
			name:   "Multiline (non-empty line)",
			fields: fields{line: &cursorMultiline, pos: 17},
			want:   16,
		},
		{
			name:   "Multiline (empty line, no move)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len() - 21},
			want:   cursorMultiline.Len() - 21,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.BeginningOfLine()

			if cur.pos != test.want {
				t.Errorf("Cursor.BeginningOfLine(): %d, should be %d", cur.pos, test.want)
			}
		})
	}
}

func TestCursor_EndOfLine(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Single line",
			fields: fields{line: &cursorLine, pos: 5},
			want:   cursorLine.Len() - 1,
		},
		{
			name:   "Multiline (non-empty line)",
			fields: fields{line: &cursorMultiline, pos: 0},
			want:   14,
		},
		{
			name:   "Multiline (empty line, no move)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len() - 21},
			want:   cursorMultiline.Len() - 21,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.EndOfLine()

			if cur.pos != test.want {
				t.Errorf("Cursor.EndOfLine(): %d, should be %d", cur.pos, test.want)
			}
		})
	}
}

func TestCursor_EndOfLineAppend(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Single line",
			fields: fields{line: &cursorLine, pos: 5},
			want:   cursorLine.Len(),
		},
		{
			name:   "Multiline (non-empty line)",
			fields: fields{line: &cursorMultiline, pos: 0},
			want:   15,
		},
		{
			name:   "Multiline (empty line, no move)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len() - 21},
			want:   cursorMultiline.Len() - 21,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.EndOfLineAppend()

			if cur.pos != test.want {
				t.Errorf("Cursor.EndOfLineAppend(): %d, should be %d", cur.pos, test.want)
			}
		})
	}
}

func TestCursor_SetMark(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name     string
		fields   fields
		expected int
	}{
		{
			name:     "Set Mark",
			fields:   fields{line: &cursorLine, pos: 10},
			expected: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.SetMark()

			if cur.pos != test.expected {
				t.Errorf("Mark: %d, should be %d", cur.mark, test.expected)
			}

			if cur.pos != cur.mark {
				t.Errorf("Cpos: %d should be equal to mark: %d", cur.pos, cur.mark)
			}
		})
	}
}

func TestCursor_Mark(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name     string
		fields   fields
		expected int
	}{
		{
			name:     "Get Mark",
			fields:   fields{line: &cursorLine, mark: 10},
			expected: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			if c.Mark() != test.expected {
				t.Errorf("Mark: %d, should be %d", c.Mark(), test.expected)
			}
		})
	}
}

func TestCursor_LinePos(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Single line",
			fields: fields{line: &cursorLine, pos: 10},
			want:   0,
		},
		{
			name:   "Multiline (second line)",
			fields: fields{line: &cursorMultiline, pos: 20},
			want:   1, // Second line.
		},
		{
			name:   "Multiline (last line, eol)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len() - 1},
			want:   len(strings.Split(string(cursorMultiline), "\n")) - 1,
		},
		{
			name:   "Multiline (last line, append-mode)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len()},
			want:   len(strings.Split(string(cursorMultiline), "\n")) - 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			if got := c.LinePos(); got != test.want {
				t.Errorf("Cursor.Line() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_LineMove(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	type args struct {
		offset int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantLine int
		wantPos  int
	}{
		{
			name:     "Single line down (on non-multiline)",
			fields:   fields{line: &cursorLine, pos: 0},
			args:     args{1},
			wantLine: 0,
			wantPos:  0,
		},
		{
			name:     "Single line down",
			fields:   fields{line: &cursorMultiline, pos: 0},
			args:     args{1},
			wantLine: 1,
			wantPos:  16,
		},
		{
			name:     "Single line up (lands on empty line)",
			fields:   fields{line: &cursorMultiline, pos: len(cursorMultiline) - 1}, // end of last line
			args:     args{-1},
			wantLine: len(strings.Split(string(cursorMultiline), "\n")) - 2,
			wantPos:  60,
		},
		{
			name:     "Out of range line up",
			fields:   fields{line: &cursorMultiline, pos: 61}, // beginning of last line
			args:     args{-5},
			wantLine: 0,
			wantPos:  0,
		},
		{
			name:     "Out of range line down",
			fields:   fields{line: &cursorMultiline, pos: 15}, // end of first line
			args:     args{5},
			wantLine: 3,
			wantPos:  61, // Since the before-last line is empty, the next move down is at the beginning of the last line.
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			c.LineMove(test.args.offset)

			if c.Pos() != test.wantPos {
				t.Errorf("Cursor: %d, want %d", c.Pos(), test.wantPos)
			}
		})
	}
}

func TestCursor_OnEmptyLine(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Empty line",
			fields: fields{line: new(Line)},
			want:   true,
		},
		{
			name:   "Multiline (empty line)",
			fields: fields{line: &cursorMultiline, pos: 60},
			want:   true,
		},
		{
			name:   "Multiline (non-empty line)",
			fields: fields{line: &cursorMultiline, pos: 61},
			want:   false,
		},
		{
			name:   "Multiline (non-empty line, append-mode)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len()},
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			if got := c.OnEmptyLine(); got != test.want {
				t.Errorf("Cursor.OnEmptyLine() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_AtBeginningOfLine(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Empty line",
			fields: fields{line: new(Line)},
			want:   true,
		},
		{
			name:   "Multiline (empty line)",
			fields: fields{line: &cursorMultiline, pos: 60},
			want:   true,
		},
		{
			name:   "Multiline (non-empty line) (at beginning)",
			fields: fields{line: &cursorMultiline, pos: 61},
			want:   true,
		},
		{
			name:   "Multiline (non-empty line) (not beginning)",
			fields: fields{line: &cursorMultiline, pos: 62},
			want:   false,
		},
		{
			name:   "Multiline (non-empty line, append-mode)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len()},
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			if got := cur.AtBeginningOfLine(); got != test.want {
				t.Errorf("Cursor.AtBeginningOfLine() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCursor_AtEndOfLine(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Empty line",
			fields: fields{line: new(Line)},
			want:   true,
		},
		{
			name:   "Multiline (empty line)",
			fields: fields{line: &cursorMultiline, pos: 59},
			want:   true,
		},
		{
			name:   "Multiline (non-empty line) (at end)",
			fields: fields{line: &cursorMultiline, pos: 58},
			want:   true,
		},
		{
			name:   "Multiline (non-empty line) (not end)",
			fields: fields{line: &cursorMultiline, pos: 57},
			want:   false,
		},
		{
			name:   "Multiline (non-empty line, append-mode)",
			fields: fields{line: &cursorMultiline, pos: cursorMultiline.Len()},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cursor{
				pos:  tt.fields.pos,
				mark: tt.fields.mark,
				line: tt.fields.line,
			}
			if got := c.AtEndOfLine(); got != tt.want {
				t.Errorf("Cursor.AtEndOfLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCursor_CheckAppend(t *testing.T) {
	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name     string
		fields   fields
		want     int
		wantMark int
	}{
		{
			name:     "Check with valid position",
			fields:   fields{line: &cursorLine, pos: 10, mark: -2},
			want:     10,
			wantMark: -1,
		},
		{
			name:     "Check with out-of-range position",
			fields:   fields{line: &cursorMultiline, pos: len(cursorMultiline) + 10, mark: 3},
			want:     len(cursorMultiline),
			wantMark: 3,
		},
		{
			name:     "Check with negative position",
			fields:   fields{line: &cursorMultiline, pos: -1, mark: cursorMultiline.Len()},
			want:     0,
			wantMark: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}
			cur.CheckAppend()

			if got := cur.Pos(); got != test.want {
				t.Errorf("Cursor.Pos() = %v, want %v", got, test.want)
			}

			if gotMark := cur.Mark(); gotMark != test.wantMark {
				t.Errorf("Cursor.Pos() = %v, want %v", gotMark, test.wantMark)
			}
		})
	}
}

func TestCursor_Coordinates(t *testing.T) {
	indent := 2 // Assumes the prompt strings uses two columns

	// Reassign the function for getting the terminal width to a fixed value
	getTermWidth = func() int { return 80 }

	type fields struct {
		pos  int
		mark int
		line *Line
	}
	tests := []struct {
		name   string
		fields fields
		wantX  int
		wantY  int
	}{
		{
			name:   "Cursor at end of buffer",
			fields: fields{line: &cursorMultiline, pos: len(cursorMultiline) - 1},
			wantX:  indent + 19,
			wantY:  len(strings.Split(string(cursorMultiline), "\n")) - 1,
		},
		{
			name:   "Cursor at beginning of buffer",
			fields: fields{line: &cursorMultiline, pos: 0},
			wantX:  indent,
			wantY:  0,
		},
		{
			name:   "Cursor on empty line",
			fields: fields{line: &cursorMultiline, pos: 60},
			wantX:  indent,
			wantY:  len(strings.Split(string(cursorMultiline), "\n")) - 2,
		},
		{
			name:   "Cursor at end of line",
			fields: fields{line: &cursorMultiline, pos: 58},
			wantX:  indent + 42,
			wantY:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Cursor{
				pos:  test.fields.pos,
				mark: test.fields.mark,
				line: test.fields.line,
			}

			gotX, gotY := CoordinatesCursor(c, indent)
			if gotX != test.wantX {
				t.Errorf("Cursor.Coordinates() gotX = %v, want %v", gotX, test.wantX)
			}

			if gotY != test.wantY {
				t.Errorf("Cursor.Coordinates() gotY = %v, want %v", gotY, test.wantY)
			}
		})
	}
}
