package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/klauspost/compress/flate"
	"github.com/muesli/termenv"
)

var (
	ErrFuncHasNotEnoughParams = errors.New("func has not enough params")
)

var (
	Debug     logs.Level = 10
	Warn      logs.Level = 20
	Info      logs.Level = 30
	Error     logs.Level = 40
	Important logs.Level = 50

	DefaultLogStyle = map[logs.Level]string{
		Debug:     termenv.String(tui.Rocket+"[+]").Bold().Background(tui.Blue).String() + " %s ",
		Warn:      termenv.String(tui.Zap+"[warn]").Bold().Background(tui.Yellow).String() + " %s ",
		Important: termenv.String(tui.Fire+"[*]").Bold().Background(tui.Purple).String() + " %s ",
		Info:      termenv.String(tui.HotSpring+"[i]").Bold().Background(tui.Green).String() + " %s ",
		Error:     termenv.String(tui.Monster+"[-]").Bold().Background(tui.Red).String() + " %s ",
	}
)

// DeflateBuf - Deflate a buffer using BestCompression (9)
func DeflateBuf(data []byte) []byte {
	var buf bytes.Buffer
	flateWriter, _ := flate.NewWriter(&buf, flate.BestCompression)
	flateWriter.Write(data)
	flateWriter.Close()
	return buf.Bytes()
}

// ByteCountBinary - Pretty print byte size
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// From the x/exp source code - gets a slice of keys for a map
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}

	return r
}

func GetParam[T any](param interface{}) (T, error) {
	val, ok := param.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("parameter type mismatch")
	}
	return val, nil
}

func MustGetParam[T any](param interface{}) T {
	val, ok := param.(T)
	if !ok {
		var zero T
		return zero
	}
	return val
}
