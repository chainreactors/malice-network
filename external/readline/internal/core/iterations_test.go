package core

import (
	"fmt"
	"testing"

	"github.com/reeflective/readline/internal/color"
)

func TestIterations_Add(t *testing.T) {
	type fields struct {
		times   string
		active  bool
		pending bool
	}
	type args struct {
		times string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantTimes   string
		wantActive  bool
		wantPending bool
	}{
		{
			name:   "Add an empty string as iterations",
			fields: fields{},
			args:   args{times: ""},
		},
		{
			name:        "Add a minus sign as iterations",
			fields:      fields{},
			args:        args{times: "-"},
			wantTimes:   "-",
			wantActive:  true,
			wantPending: true,
		},
		{
			name:        "Add a string of zeros as iterations",
			fields:      fields{},
			args:        args{times: "000"},
			wantTimes:   "000",
			wantActive:  true,
			wantPending: true,
		},
		{
			name:        "Add a minus sign to non-0 iterations",
			fields:      fields{times: "10"},
			args:        args{times: "-"},
			wantTimes:   "-10",
			wantActive:  true,
			wantPending: true,
		},
		{
			name:        "Add a negative number to iterations",
			fields:      fields{times: "10"},
			args:        args{times: "-1"},
			wantTimes:   "-101",
			wantActive:  true,
			wantPending: true,
		},
		{
			name:      "Add a string of letters to iterations (invalid)",
			fields:    fields{times: "10"},
			args:      args{times: "abc"},
			wantTimes: "10",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			iter := &Iterations{
				times:   test.fields.times,
				active:  test.fields.active,
				pending: test.fields.pending,
			}
			iter.Add(test.args.times)

			if wantTimes := test.wantTimes; iter.times != wantTimes {
				t.Errorf("Iterations.Add() = %v, want %v", iter.times, wantTimes)
			}
			if wantActive := test.wantActive; iter.active != wantActive {
				t.Errorf("Iterations.Add() = %v, want %v", iter.active, wantActive)
			}
			if wantPending := test.wantPending; iter.pending != wantPending {
				t.Errorf("Iterations.Add() = %v, want %v", iter.pending, wantPending)
			}
		})
	}
}

func TestIterations_Get(t *testing.T) {
	type fields struct {
		times   string
		active  bool
		pending bool
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Empty string is one time",
			fields: fields{},
			want:   1,
		},
		{
			name:   "Minus sign alone (-1)",
			fields: fields{times: "-"},
			want:   -1,
		},
		{
			name:   "String of zeros (000) (1)",
			fields: fields{times: "000"},
			want:   1,
		},
		{
			name:   "String of negative zeros (-000) (-1)",
			fields: fields{times: "-000"},
			want:   -1,
		},
		{
			name:   "Positive number (10) (10)",
			fields: fields{times: "10"},
			want:   10,
		},
		{
			name:   "Negative number (-10) (-10)",
			fields: fields{times: "-10"},
			want:   -10,
		},
		{
			name:   "Letters, invalid (1)",
			fields: fields{times: "abc"},
			want:   1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			iter := &Iterations{
				times:   test.fields.times,
				active:  test.fields.active,
				pending: test.fields.pending,
			}

			if got := iter.Get(); got != test.want {
				t.Errorf("Iterations.Get() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIterations_Reset(t *testing.T) {
	type fields struct {
		times   string
		active  bool
		pending bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "Empty string is one time",
			fields: fields{},
		},
		{
			name:   "Minus sign alone (-1)",
			fields: fields{times: "-"},
		},
		{
			name:   "String of zeros (000) (1)",
			fields: fields{times: "000"},
		},
		{
			name:   "String of negative zeros (-000) (-1)",
			fields: fields{times: "-000"},
		},
		{
			name:   "Positive number (10) (10)",
			fields: fields{times: "10"},
		},
		{
			name:   "Negative number (-10) (-10)",
			fields: fields{times: "-10"},
		},
		{
			name:   "Letters, invalid (1)",
			fields: fields{times: "abc"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			iter := &Iterations{
				times:   test.fields.times,
				active:  test.fields.active,
				pending: test.fields.pending,
			}
			iter.Reset()

			if got := iter.Get(); got != 1 {
				t.Errorf("Iterations.Reset() = %v, want %v", got, 1)
			}
		})
	}
}

func TestResetPostRunIterations(t *testing.T) {
	type args struct {
		iter *Iterations
	}
	type fields struct {
		times   string
		pending bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantHint string
	}{
		{
			name:   "Minus sign alone (-1) (not pending)",
			fields: fields{times: "-"},
		},
		{
			name:     "String of zeros (000) (1) (pending)",
			fields:   fields{times: "000", pending: true},
			wantHint: color.Dim + fmt.Sprintf("(arg: %s)", "000"),
		},
		{
			name:   "String of negative zeros (-000) (-1) (not pending)",
			fields: fields{times: "-000"},
		},
		{
			name:     "Positive number (10) (10) (pending)",
			fields:   fields{times: "10", pending: true},
			wantHint: color.Dim + fmt.Sprintf("(arg: %s)", "10"),
		},
		{
			name:   "Negative number (-10) (-10) (not pending)",
			fields: fields{times: "-10"},
		},
		{
			name:     "Letters, invalid (1) (pending)",
			fields:   fields{times: "abc", pending: true},
			wantHint: color.Dim + fmt.Sprintf("(arg: %s)", ""),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up the iterations
			test.args.iter = &Iterations{
				pending: test.fields.pending,
			}

			// Call the iterations add method
			test.args.iter.Add(test.fields.times)
			test.args.iter.pending = test.fields.pending

			if gotHint := ResetPostRunIterations(test.args.iter); gotHint != test.wantHint {
				t.Errorf("ResetPostRunIterations() = %v, want %v", gotHint, test.wantHint)
			}
		})
	}
}
