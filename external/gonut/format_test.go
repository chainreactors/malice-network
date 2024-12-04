package gonut

import (
	"encoding/hex"
	"testing"
)

func TestFormatTemplate_ToBase64(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(300))
	t.Logf("\n%s\n", tpl.ToBase64())
}

func TestFormatTemplate_ToBinary(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(300))
	t.Logf("\n%s\n", hex.Dump(tpl.ToBinary()))
}

func TestFormatTemplate_ToCSharp(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(276))
	t.Logf("\n%s\n", tpl.ToCSharp())
}

func TestFormatTemplate_ToHex(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(300))
	t.Logf("\n%s\n", tpl.ToHex())
}

func TestFormatTemplate_ToPowerShell(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(33))
	t.Logf("\n%s\n", tpl.ToPowerShell())
}

func TestFormatTemplate_ToPython(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(300))
	t.Logf("\n%s\n", tpl.ToPython())
}

func TestFormatTemplate_ToRubyC(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(280))
	t.Logf("\n%s\n", tpl.ToRubyC())
}

func TestFormatTemplate_ToGolang(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(277))
	t.Logf("\n%s\n", tpl.ToGolang())
}

func TestFormatTemplate_ToRust(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(277))
	t.Logf("\n%s\n", tpl.ToRust())
}

func TestFormatTemplate_ToUUID(t *testing.T) {
	tpl := NewFormatTemplate(GenRandomBytes(277))
	t.Logf("\n%s\n", tpl.ToUUID())
}
