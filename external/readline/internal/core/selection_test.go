package core

import (
	"reflect"
	"testing"
	"unicode"
)

type fields struct {
	Type       string
	active     bool
	visual     bool
	visualLine bool
	bpos       int
	epos       int
	kpos       int
	kmpos      int
	fg         string
	bg         string
	surrounds  []Selection
	line       *Line
	cursor     *Cursor
}

func TestNewSelection(t *testing.T) {
	line := Line("git command")
	cursor := NewCursor(&line)

	type args struct {
		line   *Line
		cursor *Cursor
	}
	tests := []struct {
		name string
		args args
		want *Selection
	}{
		{
			args: args{line: &line, cursor: cursor},
			want: &Selection{bpos: -1, epos: -1, line: &line, cursor: cursor},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSelection(tt.args.line, tt.args.cursor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSelection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelection_Mark(t *testing.T) {
	line, cur := newLine("multiple-ambiguous 10.203.23.45")
	cur.Set(19)

	type args struct {
		pos int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Valid position (current cursor)",
			fields:   fieldsWith(line, &cur),
			args:     args{cur.Pos()},
			wantBpos: cur.Pos(),
			wantEpos: cur.Pos(), // No movement, so both
		},
		{
			name:     "Invalid position (out of range)",
			fields:   fieldsWith(line, &cur),
			args:     args{line.Len() + 1},
			wantBpos: -1,
			wantEpos: -1,
		},
		{
			name:     "Invalid position (negative)",
			fields:   fieldsWith(line, &cur),
			args:     args{line.Len() + 1},
			wantBpos: -1,
			wantEpos: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			sel.Mark(test.args.pos)

			bpos, epos := sel.Pos()

			if bpos != test.wantBpos {
				t.Errorf("Bpos: '%d', want '%d'", bpos, test.wantBpos)
			}

			if epos != test.wantEpos {
				t.Errorf("Epos: '%d', want '%d'", epos, test.wantEpos)
			}
		})
	}
}

func TestSelection_MarkMove(t *testing.T) {
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	cur.Set(19)

	type args struct {
		pos int
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantBpos   int
		wantEpos   int
		moveCursor int
	}{
		{
			name:       "Cursor backward (valid move)",
			fields:     fieldsWith(line, &cur),
			args:       args{cur.Pos()},
			moveCursor: -2,
			wantBpos:   cur.Pos() - 2,
			wantEpos:   cur.Pos(),
		},
		{
			name:       "Cursor forward (valid move)",
			fields:     fieldsWith(line, &cur),
			args:       args{cur.Pos()},
			moveCursor: 12,
			wantBpos:   cur.Pos(),
			wantEpos:   29,
		},
		{
			name:       "Cursor to end of line",
			fields:     fieldsWith(line, &cur),
			args:       args{cur.Pos()},
			moveCursor: line.Len() - cur.Pos(),
			wantBpos:   cur.Pos(),
			wantEpos:   line.Len(),
		},
		{
			name:     "Cursor out-of-range move (checked)",
			fields:   fieldsWith(line, &cur),
			args:     args{cur.Pos()},
			wantBpos: cur.Pos(),
			wantEpos: line.Len(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			// Mark and move the cursor
			sel.Mark(test.args.pos)
			cur.Move(test.moveCursor)

			bpos, epos := sel.Pos()

			if bpos != test.wantBpos {
				t.Errorf("Bpos: '%d', want '%d'", bpos, test.wantBpos)
			}

			if epos != test.wantEpos {
				t.Errorf("Epos: '%d', want '%d'", epos, test.wantEpos)
			}
		})
	}
}

func TestSelection_MarkRange(t *testing.T) {
	line, cur := newLine("multiple-ambiguous 10.203.23.45")
	pos := cur.Pos()

	type args struct {
		bpos       int
		epos       int
		moveCursor int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Valid range (cursor to end of line)",
			fields:   fieldsWith(line, &cur),
			args:     args{cur.Pos(), line.Len(), 0},
			wantBpos: cur.Pos(),
			wantEpos: line.Len(),
		},
		{
			name:     "Invalid range (both positive out-of-line values)",
			fields:   fieldsWith(line, &cur),
			args:     args{line.Len() + 1, line.Len() + 10, 0},
			wantBpos: -1,
			wantEpos: -1,
		},
		{
			name:     "Range with negative epos (uses cursor pos instead)",
			fields:   fieldsWith(line, &cur),
			args:     args{cur.Pos(), -1, 5},
			wantBpos: cur.Pos(),
			wantEpos: cur.Pos() + 5,
		},
		{
			name:     "Range with negative bpos (uses cursor pos instead)",
			fields:   fieldsWith(line, &cur),
			args:     args{-1, cur.Pos(), 5},
			wantBpos: cur.Pos(),
			wantEpos: cur.Pos() + 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cur.Set(pos)

			sel := newTestSelection(test.fields)

			sel.MarkRange(test.args.bpos, test.args.epos)
			cur.Move(test.args.moveCursor)

			bpos, epos := sel.Pos()

			if bpos != test.wantBpos {
				t.Errorf("Bpos: '%d', want '%d'", bpos, test.wantBpos)
			}

			if epos != test.wantEpos {
				t.Errorf("Epos: '%d', want '%d'", epos, test.wantEpos)
			}
		})
	}
}

func TestSelection_MarkSurround(t *testing.T) {
	line, cur := newLine("multiple-ambiguous '10.203.23.45 127.0.0.1' ::1")
	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantSelections int
		wantBpos       int
		wantEpos       int
		wantBposS2     int
		wantEposS2     int
	}{
		{
			name:           "Valid surround (single quotes)",
			fields:         fieldsWith(line, &cur),
			args:           args{19, 42},
			wantSelections: 2,
			wantBpos:       19,
			wantEpos:       20,
			wantBposS2:     42,
			wantEposS2:     43,
		},
		{
			name:           "Invalid (epos out of range)",
			fields:         fieldsWith(line, &cur),
			args:           args{19, line.Len() + 1},
			wantSelections: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			sel.MarkSurround(test.args.bpos, test.args.epos)

			if len(sel.Surrounds()) != test.wantSelections {
				t.Errorf("Surround areas: '%d', want '%d'", len(sel.Surrounds()), test.wantSelections)
			}

			if len(sel.Surrounds()) == 0 {
				return
			}

			// Surround 1
			bpos, epos := sel.Surrounds()[0].Pos()
			if bpos != test.wantBpos {
				t.Errorf("Bpos: '%d', want '%d'", bpos, test.wantBpos)
			}

			if epos != test.wantEpos {
				t.Errorf("Epos: '%d', want '%d'", epos, test.wantEpos)
			}

			// Surround 2
			bpos, epos = sel.Surrounds()[1].Pos()
			if bpos != test.wantBposS2 {
				t.Errorf("BposS2: '%d', want '%d'", bpos, test.wantBposS2)
			}

			if epos != test.wantEposS2 {
				t.Errorf("EposS2: '%d', want '%d'", epos, test.wantEposS2)
			}
		})
	}
}

func TestSelection_Active(t *testing.T) {
	line, cur := newLine("multiple-ambiguous 10.203.23.45")
	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "Valid position (current cursor)",
			fields: fieldsWith(line, &cur),
			args:   args{cur.Pos(), cur.Pos() + 1},
			want:   true,
		},
		{
			name:   "Invalid range (both positive out-of-line values)",
			fields: fieldsWith(line, &cur),
			args:   args{line.Len() + 1, line.Len() + 10},
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			sel.MarkRange(test.args.bpos, test.args.epos)

			if got := sel.Active(); got != test.want {
				t.Errorf("Selection.Active() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSelection_Visual(t *testing.T) {
	single, cSingle := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	multi, cMulti := newLine("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		bpos       int
		epos       int
		cursorMove int
		visualLine bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		// Visual
		{
			name:     "Cursor position (single character)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: cSingle.Pos(), epos: -1},
			wantBpos: 0,
			wantEpos: 1,
		},
		{
			name:     "Cursor to end of line",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: cSingle.Pos(), epos: -1, cursorMove: single.Len() - cSingle.Pos()},
			wantBpos: 0,
			wantEpos: single.Len(),
		},
		// Visual line
		{
			name:     "Visual line (single line)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: 20, epos: -1, visualLine: true},
			wantBpos: 0,
			wantEpos: single.Len(),
		},
		{
			name:     "Visual line (multiline)",
			fields:   fieldsWith(multi, &cMulti),
			args:     args{bpos: 24, epos: -1, visualLine: true, cursorMove: 24},
			wantBpos: 23,
			wantEpos: 61,
		},
		{
			name:     "Visual line cursor movement (multiline)",
			fields:   fieldsWith(multi, &cMulti),
			args:     args{bpos: 24, epos: -1, visualLine: true, cursorMove: 0},
			wantBpos: 0,
			wantEpos: 61,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.fields.cursor.Set(0)

			sel := newTestSelection(test.fields)

			// First create the mark, then only after set it to visual/line
			sel.MarkRange(test.args.bpos, test.args.epos)
			sel.Visual(test.args.visualLine)

			test.fields.cursor.Move(test.args.cursorMove)

			gotBpos, gotEpos := sel.Pos()
			if gotBpos != test.wantBpos {
				t.Errorf("Selection.Pos() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.Pos() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestSelection_Pos(t *testing.T) {
	// selection.Pos() is actually used in many/all other tests in this file,
	// so this test is meant to try wrong values that could only be set internally,
	// (like invalid selection positions), thus this tests weird, unlikely internal errors.
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45")

	tests := []struct {
		name         string
		fields       fields
		override     *fields
		overrideCpos int
		wantBpos     int
		wantEpos     int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			wantBpos: -1,
			wantEpos: -1,
		},
		{
			name:     "Invalid internal positions",
			fields:   fieldsWith(line, &cur),
			override: &fields{active: true, bpos: -2, epos: -10},
			wantBpos: -1,
			wantEpos: -1,
		},
		{
			name:         "Invalid cursor position & pending selection",
			fields:       fieldsWith(line, &cur),
			override:     &fields{active: true, bpos: 10, epos: -1},
			overrideCpos: -3,
			wantBpos:     0,
			wantEpos:     10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := &Selection{
				Type:       test.fields.Type,
				active:     test.fields.active,
				visual:     test.fields.visual,
				visualLine: test.fields.visualLine,
				bpos:       test.fields.bpos,
				epos:       test.fields.epos,
				kpos:       test.fields.kpos,
				kmpos:      test.fields.kmpos,
				fg:         test.fields.fg,
				bg:         test.fields.bg,
				surrounds:  test.fields.surrounds,
				line:       test.fields.line,
				cursor:     test.fields.cursor,
			}

			// Overriding some default values to test weird internal error cases.
			if test.override != nil {
				sel.active = test.override.active
				sel.bpos = test.override.bpos
				sel.epos = test.override.epos
			}

			if test.overrideCpos != 0 {
				sel.cursor.Set(test.overrideCpos)
			}

			gotBpos, gotEpos := sel.Pos()
			if gotBpos != test.wantBpos {
				t.Errorf("Selection.Pos() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.Pos() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestSelection_Cursor(t *testing.T) {
	emptyline, emptycur := newLine("")
	single, cSingle := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	multi, cMulti := newLine("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		bpos       int
		cursorMove int
		visualLine bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantCpos int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			wantCpos: 0,
		},
		{
			name:     "Cursor forward word",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: 0, cursorMove: 6},
			wantCpos: 0, // Forward selection does not move the cursor.
		},
		{
			name:     "Cursor backward word",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: 6, cursorMove: -6},
			wantCpos: 0, // Backward selection, if deleted, would move the cursor back.
		},
		{
			name:     "Cursor on last line (single line) (visual line selection)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: single.Len(), visualLine: true},
			wantCpos: single.Len(), // Position of the cursor on the previous line if we deleted our selected line.
		},
		{
			name:     "Cursor on last line (multiline) (visual line selection)",
			fields:   fieldsWith(multi, &cMulti),
			args:     args{bpos: multi.Len() - 1, visualLine: true},
			wantCpos: 31, // Position of the cursor on the previous line if we deleted our selected line.
		},
		{
			name:   "Current line longer than next (visual line selection)",
			fields: fieldsWith(multi, &cMulti),
			args:   args{bpos: multi.Len() - 11, visualLine: true}, // end of before-last line.
			// Same here: vertical position of cursor greater than last line length, becomes last line length.
			wantCpos: 31,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			// Place the cursor where we want to start selecting.
			test.fields.cursor.Set(test.args.bpos)
			sel.Mark(test.fields.cursor.Pos())

			if test.args.visualLine {
				sel.Visual(test.args.visualLine)
			}

			// Move the cursor when needed.
			test.fields.cursor.Move(test.args.cursorMove)

			cpos := sel.Cursor()
			if cpos != test.wantCpos {
				t.Errorf("Selection.Cursor() cpos = %v, want %v", cpos, test.wantCpos)
			}
		})
	}
}

func TestSelection_Len(t *testing.T) {
	emptyline, emptycur := newLine("")
	single, cSingle := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	multi, cMulti := newLine("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		bpos       int
		cursorMove int
		visual     bool
		visualLine bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Empty line",
			fields: fieldsWith(emptyline, &emptycur),
			want:   0,
		},
		{
			name:   "Cursor forward word",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: 0, cursorMove: 6},
			want:   6,
		},
		{
			name:   "Cursor backward word",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: 6, cursorMove: -6},
			want:   6,
		},
		{
			name:   "Cursor on last line (single line) (visual line selection)",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: single.Len(), visualLine: true},
			want:   single.Len(),
		},
		{
			name:   "Visual line (multiline)",
			fields: fieldsWith(multi, &cMulti),
			args:   args{bpos: 24, visualLine: true, cursorMove: 24},
			want:   38,
		},
		{
			name:   "Visual line cursor movement (multiline)",
			fields: fieldsWith(multi, &cMulti),
			args:   args{bpos: 24, visualLine: true, cursorMove: 0},
			want:   38,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)
			sel.visual = test.args.visual

			// Place the cursor where we want to start selecting.
			test.fields.cursor.Set(test.args.bpos)
			sel.Mark(test.fields.cursor.Pos())

			if test.args.visualLine {
				sel.Visual(test.args.visualLine)
			}

			// Move the cursor when needed.
			test.fields.cursor.Move(test.args.cursorMove)

			if got := sel.Len(); got != test.want {
				t.Errorf("Selection.Len() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSelection_Text(t *testing.T) {
	emptyline, emptycur := newLine("")
	single, cSingle := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")

	type args struct {
		bpos       int
		cursorMove int
		visual     bool
		visualLine bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "Empty line",
			fields: fieldsWith(emptyline, &emptycur),
			want:   "",
		},
		{
			name:   "Cursor forward word",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: 0, cursorMove: 8},
			want:   "multiple",
		},
		{
			name:   "Cursor backward word",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: 8, cursorMove: -8},
			want:   "multiple",
		},
		{
			name:   "Cursor on last line (single line) (visual line selection)",
			fields: fieldsWith(single, &cSingle),
			args:   args{bpos: 0, visualLine: true},
			want:   string(single),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)
			sel.visual = test.args.visual

			// Place the cursor where we want to start selecting.
			test.fields.cursor.Set(test.args.bpos)
			sel.Mark(test.fields.cursor.Pos())

			if test.args.visualLine {
				sel.Visual(test.args.visualLine)
			}

			// Move the cursor when needed.
			test.fields.cursor.Move(test.args.cursorMove)

			if got := sel.Text(); got != test.want {
				t.Errorf("Selection.Text() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSelection_Pop(t *testing.T) {
	emptyline, emptycur := newLine("")
	single, cSingle := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	multi, cMulti := newLine("basic -f \"commands.go \nanother testing\" --alternate \"another\nquote\" -c")

	type args struct {
		bpos       int
		moveCursor int
		visual     bool
		visualLine bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBuf  string
		wantBpos int
		wantEpos int
		wantCpos int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			wantBpos: -1,
			wantEpos: -1,
			wantCpos: cSingle.Pos(),
		},
		// Single line
		{
			name:     "No range (not visual, no move, epos=bpos)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: cSingle.Pos()},
			wantBpos: cSingle.Pos(),
			wantEpos: cSingle.Pos(),
			wantCpos: cSingle.Pos(),
		},
		{
			name:     "Valid cursor (visual, no move)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: cSingle.Pos(), visual: true},
			wantBpos: cSingle.Pos(),
			wantEpos: cSingle.Pos() + 1,
			wantCpos: cSingle.Pos(),
			wantBuf:  "m",
		},
		{
			name:     "Valid range (cursor to end of line)",
			fields:   fieldsWith(single, &cSingle),
			args:     args{bpos: cSingle.Pos(), moveCursor: single.Len() - cSingle.Pos()},
			wantBpos: cSingle.Pos(),
			wantEpos: single.Len(),
			wantBuf:  string(single),
		},
		// Multiline
		{
			name:     "Cursor on last line (visual line selection)",
			fields:   fieldsWith(multi, &cMulti),
			args:     args{bpos: multi.Len() - 1, visualLine: true},
			wantCpos: 31, // Position of the cursor on the previous line if we deleted our selected line.
			wantBpos: 61,
			wantEpos: multi.Len(),
			wantBuf:  "quote\" -c",
		},
		{
			name:   "Current line longer than next (visual line selection)",
			fields: fieldsWith(multi, &cMulti),
			args:   args{bpos: multi.Len() - 11, visualLine: true}, // end of before-last line.
			// Same here: vertical position of cursor greater than last line length, becomes last line length.
			wantCpos: 31,
			wantBpos: 23,
			wantEpos: 61,
			wantBuf:  "another testing\" --alternate \"another\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			// Place the cursor where we want to start selecting.
			test.fields.cursor.Set(test.args.bpos)
			sel.Mark(test.fields.cursor.Pos())

			if test.args.visualLine || test.args.visual {
				sel.Visual(test.args.visualLine)
			}

			// Move the cursor when needed.
			test.fields.cursor.Move(test.args.moveCursor)

			gotBuf, gotBpos, gotEpos, gotCpos := sel.Pop()
			if test.wantBuf != "" && gotBuf != test.wantBuf {
				t.Errorf("Selection.Pop() gotBuf = %v, want %v", gotBuf, test.wantBuf)
			}

			if gotBpos != test.wantBpos {
				t.Errorf("Selection.Pop() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.Pop() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}

			if gotCpos != test.wantCpos {
				t.Errorf("Selection.Pop() gotCpos = %v, want %v", gotCpos, test.wantCpos)
			}

			// Selection should be reset.
			testSelectionReset(t, sel)
		})
	}
}

func TestSelection_InsertAt(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")

	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantBuf string
	}{
		{
			name:    "Empty line",
			fields:  fieldsWith(emptyline, &emptycur),
			wantBuf: "",
		},
		{
			name:    "Valid range insertion",
			args:    args{bpos: line.Len() - 10, epos: line.Len()}, // The line won't actually change.
			wantBuf: string(line),
		},
		{
			name:    "Invalid range insertion (epos out of bounds)",
			args:    args{bpos: line.Len() - 10, epos: line.Len() + 10}, // The line won't actually change.
			wantBuf: string(line),
		},
		{
			name:    "Insert at end of line",
			args:    args{bpos: line.Len(), epos: -1},
			wantBuf: string(line) + " 127.0.0.1",
		},
		{
			name:    "Insert at begin position (epos == -1)",
			args:    args{bpos: 18, epos: -1},
			wantBuf: "multiple-ambiguous 127.0.0.1 10.203.23.45 127.0.0.1",
		},
		{
			name:    "Insert at end position (bpos == -1)",
			args:    args{bpos: -1, epos: 18},
			wantBuf: "multiple-ambiguous 127.0.0.1 10.203.23.45 127.0.0.1",
		},
		{
			name:    "Insert at begin position (bpos == epos)",
			args:    args{bpos: 18, epos: 18},
			wantBuf: "multiple-ambiguous 127.0.0.1 10.203.23.45 127.0.0.1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, cur = newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
			if test.fields.line == nil || test.fields.line.Len() != 0 {
				test.fields.line, test.fields.cursor = &line, &cur
			}

			sel := newTestSelection(test.fields)

			// Select the last IP.
			sel.MarkRange(test.fields.line.Len()-10, test.fields.line.Len())

			// Insert according to test spec.
			sel.InsertAt(test.args.bpos, test.args.epos)

			// Check line contents and selection reset.
			gotBuf := string(*test.fields.line)
			if gotBuf != test.wantBuf {
				t.Errorf("Selection.InsertAt() gotBuf = %v, want %v", gotBuf, test.wantBuf)
			}

			testSelectionReset(t, sel)
		})
	}
}

func TestSelection_Surround(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	type args struct {
		bchar rune
		echar rune
		bpos  int
		epos  int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantBuf string
	}{
		{
			name:    "Empty line",
			fields:  fieldsWith(emptyline, &emptycur),
			args:    args{bchar: '"', echar: '"', bpos: 0, epos: 0},
			wantBuf: "",
		},
		{
			name:    "Valid range",
			fields:  fieldsWith(line, &cur),
			args:    args{bchar: '"', echar: '"', bpos: 19, epos: 19 + 12},
			wantBuf: "multiple-ambiguous \"10.203.23.45\" 127.0.0.1",
		},
		{
			name:    "Valid range (epos at end of line)",
			fields:  fieldsWith(line, &cur),
			args:    args{bchar: '\'', echar: '\'', bpos: 32, epos: line.Len()},
			wantBuf: "multiple-ambiguous 10.203.23.45 '127.0.0.1'",
		},
		{
			name:    "Invalid range (epos out of range)",
			fields:  fieldsWith(line, &cur),
			args:    args{bchar: '\'', echar: '\'', bpos: 32, epos: line.Len() + 1},
			wantBuf: "multiple-ambiguous 10.203.23.45 '127.0.0.1'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, cur = newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
			if test.fields.line == nil || test.fields.line.Len() != 0 {
				test.fields.line, test.fields.cursor = &line, &cur
			}

			sel := newTestSelection(test.fields)

			// Mark and surround the selection.
			sel.MarkRange(test.args.bpos, test.args.epos)
			sel.Surround(test.args.bchar, test.args.echar)

			// Check line contents and selection reset.
			gotBuf := string(*test.fields.line)
			if gotBuf != test.wantBuf {
				t.Errorf("Selection.Surround() gotBuf = %v, want %v", gotBuf, test.wantBuf)
			}

			testSelectionReset(t, sel)
		})
	}
}

func TestSelection_SelectAWord(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")

	type args struct {
		cpos int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			args:     args{cpos: 0},
			wantBpos: 0,
			wantEpos: 0,
		},
		{
			name:     "On space (fail)",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: 18},
			wantBpos: 18,
			wantEpos: 18,
		},
		{
			name:     "On digit",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: 19},
			wantBpos: 18,
			wantEpos: 20,
		},
		{
			name:     "On last digit of word",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: 30},
			wantBpos: 29,
			wantEpos: 30,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)
			sel.cursor.Set(test.args.cpos)

			gotBpos, gotEpos := sel.SelectAWord()
			if gotBpos != test.wantBpos {
				t.Errorf("Selection.SelectAWord() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.SelectAWord() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestSelection_SelectABlankWord(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")

	type args struct {
		cpos int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			args:     args{cpos: 0},
			wantBpos: 0,
			wantEpos: 0,
		},
		{
			name:     "On space (select following word and leading spaces)",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: 18},
			wantBpos: 18,
			wantEpos: 30,
		},
		{
			name:     "Cursor at beginning of line (with trailing spaces)",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: 0},
			wantBpos: 0,
			wantEpos: 18,
		},
		{
			name:     "Cursor at end of line (with leading spaces)",
			fields:   fieldsWith(line, &cur),
			args:     args{cpos: line.Len()},
			wantBpos: 31,
			wantEpos: line.Len() - 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)
			sel.cursor.Set(test.args.cpos)

			gotBpos, gotEpos := sel.SelectABlankWord()
			if gotBpos != test.wantBpos {
				t.Errorf("Selection.SelectABlankWord() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.SelectABlankWord() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestSelection_SelectAShellWord(t *testing.T) {
	emptyline, emptycur := newLine("")
	multiline, mCursor := newLine("git command -c \n \"second line of input\" before an empty \"line \n\n and then\" a last quoted-\"shell-word one\" and 'trailing shell'-word")

	type args struct {
		cpos int
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBpos int
		wantEpos int
	}{
		{
			name:     "Empty line",
			fields:   fieldsWith(emptyline, &emptycur),
			args:     args{cpos: 0},
			wantBpos: 0,
			wantEpos: 0,
		},

		{
			name:     "Cursor on a single word (with leading spaces)",
			fields:   fieldsWith(multiline, &mCursor),
			args:     args{cpos: 4},
			wantBpos: 3,
			wantEpos: 10,
		},
		{
			name:     "Cursor in a shell word (with leading spaces)",
			fields:   fieldsWith(multiline, &mCursor),
			args:     args{cpos: 23},
			wantBpos: 14,
			wantEpos: 38,
		},
		{
			name:     "Cursor on an empty line",
			fields:   fieldsWith(multiline, &mCursor),
			args:     args{cpos: 63},
			wantBpos: 55,
			wantEpos: 73,
		},
		{
			name:     "Cursor on mixed shell and leading blank word",
			fields:   fieldsWith(multiline, &mCursor),
			args:     args{cpos: 95},
			wantBpos: 81,
			wantEpos: 104,
		},
		{
			name:     "Cursor on mixed shell and trailing blank words",
			fields:   fieldsWith(multiline, &mCursor),
			args:     args{cpos: multiline.Len() - 1},
			wantBpos: 119,
			wantEpos: multiline.Len() - 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)
			sel.cursor.Set(test.args.cpos)

			gotBpos, gotEpos := sel.SelectAShellWord()
			if gotBpos != test.wantBpos {
				t.Errorf("Selection.SelectAShellWord() gotBpos = %v, want %v", gotBpos, test.wantBpos)
			}

			if gotEpos != test.wantEpos {
				t.Errorf("Selection.SelectAShellWord() gotEpos = %v, want %v", gotEpos, test.wantEpos)
			}
		})
	}
}

func TestSelection_SelectKeyword(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	urlLine, urlCur := newLine("git command http://domain.word.com/url?key=value&test=val,testingthis this word http://10.203.23.45:3999")

	type args struct {
		cpos   int
		bpos   int
		epos   int
		next   bool
		cycles int
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantKbpos int
		wantKepos int
		wantMatch bool
	}{
		{
			name:      "Empty line",
			fields:    fieldsWith(emptyline, &emptycur),
			args:      args{bpos: 0, epos: 0, next: true, cycles: 1},
			wantKbpos: -1,
			wantKepos: -1,
			wantMatch: false,
		},
		{
			name:      "Single line, cursor in the middle of the first IP address",
			fields:    fieldsWith(line, &cur),
			args:      args{cpos: 20, bpos: 0, epos: 0, next: true, cycles: 1},
			wantKbpos: 19,
			wantKepos: 31,
			wantMatch: false,
		},
		{
			name:      "Single line, cursor at the beginning of the third word",
			fields:    fieldsWith(urlLine, &urlCur),
			args:      args{cpos: 32, bpos: 0, epos: 0, next: true, cycles: 1},
			wantKbpos: 12,
			wantKepos: 12 + 57,
			wantMatch: true,
		},
		{
			name:      "Multiple subgroups cycle, forward",
			fields:    fieldsWith(urlLine, &urlCur),
			args:      args{cpos: 32, bpos: 0, epos: 0, next: true, cycles: 2},
			wantKbpos: 19,
			wantKepos: 12 + 57,
			wantMatch: true,
		},
		{
			name:      "Multiple subgroups cycle, reverse",
			fields:    fieldsWith(urlLine, &urlCur),
			args:      args{cpos: 32, bpos: 0, epos: 0, next: false, cycles: 2},
			wantKbpos: 39,
			wantKepos: 12 + 57,
			wantMatch: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			sel.cursor.Set(test.args.cpos)
			sel.SelectABlankWord()

			// reassign the args bpos/epos to the selection bpos/epos
			bpos, epos := sel.Pos()
			test.args.bpos = bpos
			test.args.epos = epos

			var gotKbpos, gotKepos int
			var gotMatch bool

			for i := 0; i < test.args.cycles; i++ {
				gotKbpos, gotKepos, gotMatch = sel.SelectKeyword(test.args.bpos, test.args.epos, test.args.next)
			}

			if gotKbpos != test.wantKbpos {
				t.Errorf("Selection.SelectKeyword() gotKbpos = %v, want %v", gotKbpos, test.wantKbpos)
			}

			if gotKepos != test.wantKepos {
				t.Errorf("Selection.SelectKeyword() gotKepos = %v, want %v", gotKepos, test.wantKepos)
			}

			if gotMatch != test.wantMatch {
				t.Errorf("Selection.SelectKeyword() gotMatch = %v, want %v", gotMatch, test.wantMatch)
			}
		})
	}
}

func TestSelection_ReplaceWith(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous lower UPPER")

	type args struct {
		bpos     int
		epos     int
		replacer func(r rune) rune
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantBuf string
	}{
		{
			name:    "Empty line",
			fields:  fieldsWith(emptyline, &emptycur),
			args:    args{bpos: 0, epos: 0, replacer: unicode.ToLower},
			wantBuf: "",
		},
		{
			name:    "Replace to upper",
			fields:  fieldsWith(line, &cur),
			args:    args{bpos: 19, epos: 24, replacer: unicode.ToUpper},
			wantBuf: "multiple-ambiguous LOWER UPPER",
		},
		{
			name:    "Replace to lower (with epos out-of-range)",
			fields:  fieldsWith(line, &cur),
			args:    args{bpos: 25, epos: line.Len() + 1, replacer: unicode.ToLower},
			wantBuf: "multiple-ambiguous lower upper",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, cur = newLine("multiple-ambiguous lower UPPER")
			if test.fields.line == nil || test.fields.line.Len() != 0 {
				test.fields.line, test.fields.cursor = &line, &cur
			}

			sel := newTestSelection(test.fields)

			// Mark and replace the selection.
			sel.MarkRange(test.args.bpos, test.args.epos)
			sel.ReplaceWith(test.args.replacer)

			// Check line contents and selection reset.
			gotBuf := string(*test.fields.line)
			if gotBuf != test.wantBuf {
				t.Errorf("Selection.ReplaceWith() gotBuf = %v, want %v", gotBuf, test.wantBuf)
			}

			testSelectionReset(t, sel)
		})
	}
}

func TestSelection_Cut(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous 10.203.23.45 127.0.0.1")
	sline, scur := newLine("multiple-ambiguous '10.203.23.45' 127.0.0.1")
	multiline, mCursor := newLine("git command -c \n second line of input before an empty line \n\n and then a last one")

	type args struct {
		bpos       int
		epos       int
		visualLine bool
		selectFunc func(*Selection)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantCut string
		wantBuf string
	}{
		{
			name:    "Empty line",
			fields:  fieldsWith(emptyline, &emptycur),
			args:    args{bpos: 0, epos: 0},
			wantCut: "",
		},
		{
			name:    "Single line, cursor at beginning of line (blank word)",
			fields:  fieldsWith(line, &cur),
			args:    args{bpos: 0, epos: 0, selectFunc: func(s *Selection) { cur.BeginningOfLine(); s.SelectABlankWord() }},
			wantCut: "multiple-ambiguous",
			wantBuf: " 10.203.23.45 127.0.1",
		},
		{
			name:    "Multiline, cursor after first newline, visualLine true",
			fields:  fieldsWith(multiline, &mCursor),
			args:    args{bpos: 20, epos: -1, visualLine: true},
			wantCut: " second line of input before an empty line \n",
			wantBuf: "git command -c \n\n and then a last one",
		},
		{
			name:    "Single line, cursor in the middle of an IP address",
			fields:  fieldsWith(sline, &scur),
			args:    args{bpos: 22, epos: -1, selectFunc: func(s *Selection) { s.MarkSurround(19, 31) }},
			wantCut: "",
			wantBuf: "multiple-ambiguous 10.203.23.45 127.0.0.1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := NewSelection(test.fields.line, test.fields.cursor)

			if test.args.epos == -1 {
				test.fields.cursor.Set(test.args.bpos)
			}

			// either use the selection function or fixed positions.
			if test.args.selectFunc != nil {
				test.args.selectFunc(sel)
			} else {
				sel.MarkRange(test.args.bpos, test.args.epos)
			}

			if test.args.visualLine {
				sel.Visual(true)
			}

			if gotBuf := sel.Cut(); gotBuf != test.wantCut {
				t.Errorf("Selection.Cut() = %v, want %v", gotBuf, test.wantCut)
			}
		})
	}
}

func TestHighlightMatchers(t *testing.T) {
	emptyline, emptycur := newLine("")
	line, cur := newLine("multiple-ambiguous { surrounded 'quoted word' } words")

	type args struct {
		cpos int
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantSurrounds int
	}{
		{
			name:          "Empty line",
			fields:        fieldsWith(emptyline, &emptycur),
			args:          args{cpos: 0},
			wantSurrounds: 0,
		},
		{
			name:          "Cursor on opening token",
			fields:        fieldsWith(line, &cur),
			args:          args{cpos: 19},
			wantSurrounds: 1,
		},
		{
			name:          "Cursor on closing token",
			fields:        fieldsWith(line, &cur),
			args:          args{cpos: 46},
			wantSurrounds: 1,
		},
		{
			name:          "Cursor not on token",
			fields:        fieldsWith(line, &cur),
			args:          args{cpos: 25},
			wantSurrounds: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			test.fields.cursor.Set(test.args.cpos)
			HighlightMatchers(sel)

			if len(sel.surrounds) != test.wantSurrounds {
				t.Errorf("ResetMatchers() len(sel.surrounds) = %v, want %v", len(sel.surrounds), test.wantSurrounds)
			}
		})
	}
}

func TestResetMatchers(t *testing.T) {
	line, cur := newLine("multiple-ambiguous { surrounded 'quoted word' } words")
	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantActive     bool
		wantVisual     bool
		wantVisualLine bool
		wantBpos       int
		wantEpos       int
		wantSurrounds  int
	}{
		{
			name:           "Select and reset",
			fields:         fieldsWith(line, &cur),
			args:           args{bpos: 32, epos: 44},
			wantActive:     true,
			wantBpos:       -1,
			wantEpos:       -1,
			wantVisual:     false,
			wantVisualLine: false,
			wantSurrounds:  2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			// Add some surround blinking matchers to the selection
			test.fields.cursor.Set(19)
			HighlightMatchers(sel)

			if len(sel.surrounds) != 1 {
				t.Errorf("ResetMatchers() len(sel.surrounds) = %v, want %v", len(sel.surrounds), test.wantSurrounds)
			}

			// Surround select the quotes
			sel.MarkSurround(test.args.bpos, test.args.epos)
			ResetMatchers(sel)

			if sel.active != test.wantActive {
				t.Errorf("ResetMatchers() sel.active = %v, want %v", sel.active, test.wantActive)
			}

			if sel.bpos != test.wantBpos {
				t.Errorf("ResetMatchers() sel.bpos = %v, want %v", sel.bpos, test.wantBpos)
			}

			if sel.epos != test.wantEpos {
				t.Errorf("ResetMatchers() sel.epos = %v, want %v", sel.epos, test.wantEpos)
			}

			if sel.visual != test.wantVisual {
				t.Errorf("ResetMatchers() sel.visual = %v, want %v", sel.visual, test.wantVisual)
			}

			if sel.visualLine != test.wantVisualLine {
				t.Errorf("ResetMatchers() sel.visualLine = %v, want %v", sel.visualLine, test.wantVisualLine)
			}

			if len(sel.surrounds) != test.wantSurrounds {
				t.Errorf("ResetMatchers() len(sel.surrounds) = %v, want %v", len(sel.surrounds), test.wantSurrounds)
			}
		})
	}
}

func TestSelection_Reset(t *testing.T) {
	line, cur := newLine("multiple-ambiguous {surrounded test} words")
	type args struct {
		bpos int
		epos int
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantActive     bool
		wantVisual     bool
		wantVisualLine bool
		wantBpos       int
		wantEpos       int
		wantSurrounds  int
		wantFg         string
		wantBg         string
	}{
		{
			name:           "Select and reset",
			fields:         fieldsWith(line, &cur),
			args:           args{bpos: 0, epos: line.Len()},
			wantActive:     false,
			wantBpos:       -1,
			wantEpos:       -1,
			wantVisual:     false,
			wantVisualLine: false,
			wantFg:         "",
			wantBg:         "",
			wantSurrounds:  1, // One blinking matcher
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sel := newTestSelection(test.fields)

			// Add some surround blinking matchers to the selection
			test.fields.cursor.Set(19)
			HighlightMatchers(sel)

			// Mark the selection and reset it.
			sel.MarkRange(test.args.bpos, test.args.epos)
			sel.Reset()

			if sel.active != test.wantActive {
				t.Errorf("Selection.Reset() sel.active = %v, want %v", sel.active, test.wantActive)
			}

			if sel.bpos != test.wantBpos {
				t.Errorf("Selection.Reset() sel.bpos = %v, want %v", sel.bpos, test.wantBpos)
			}

			if sel.epos != test.wantEpos {
				t.Errorf("Selection.Reset() sel.epos = %v, want %v", sel.epos, test.wantEpos)
			}

			if sel.visual != test.wantVisual {
				t.Errorf("Selection.Reset() sel.visual = %v, want %v", sel.visual, test.wantVisual)
			}

			if sel.visualLine != test.wantVisualLine {
				t.Errorf("Selection.Reset() sel.visualLine = %v, want %v", sel.visualLine, test.wantVisualLine)
			}

			if sel.fg != test.wantFg {
				t.Errorf("Selection.Reset() sel.fg = %v, want %v", sel.fg, test.wantFg)
			}

			if sel.bg != test.wantBg {
				t.Errorf("Selection.Reset() sel.bg = %v, want %v", sel.bg, test.wantBg)
			}

			if len(sel.surrounds) != test.wantSurrounds {
				t.Errorf("Selection.Reset() len(sel.surrounds) = %v, want %v", len(sel.surrounds), test.wantSurrounds)
			}
		})
	}
}

func testSelectionReset(t *testing.T, sel *Selection) {
	t.Helper()

	if sel.Text() != "" {
		t.Errorf("Selection.Reset() gotBuf = %v, want %v", sel.Text(), "")
	}

	if sel.bpos != -1 {
		t.Errorf("Selection.Reset() gotBpos = %v, want %v", sel.bpos, -1)
	}

	if sel.epos != -1 {
		t.Errorf("Selection.Reset() epos = %v, want %v", sel.epos, -1)
	}

	if sel.active {
		t.Error("Selection.Reset() is still active, should not be")
	}
}

//
// Helpers ------------------------------------------------
//

func newLine(line string) (Line, Cursor) {
	l := Line([]rune(line))
	c := Cursor{line: &l}

	return l, c
}

func fieldsWith(l Line, c *Cursor) fields {
	return fields{
		line:   &l,
		cursor: c,
		bpos:   -1,
		epos:   -1,
		Type:   "visual",
	}
}

func newTestSelection(fields fields) *Selection {
	return &Selection{
		Type:      fields.Type,
		active:    fields.active,
		bpos:      fields.bpos,
		epos:      fields.epos,
		fg:        fields.fg,
		bg:        fields.bg,
		surrounds: fields.surrounds,
		line:      fields.line,
		cursor:    fields.cursor,
	}
}
