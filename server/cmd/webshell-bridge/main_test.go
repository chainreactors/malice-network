package main

import (
	"io"
	"os"
	"testing"

	"github.com/chainreactors/logs"
)

func TestMain(m *testing.M) {
	logs.Log = logs.NewLogger(logs.WarnLevel)
	logs.Log.SetOutput(io.Discard)
	os.Exit(m.Run())
}
