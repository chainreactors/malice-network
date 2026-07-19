package core

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewConsolePreservesLiteralBackslashes(t *testing.T) {
	client := &Console{}
	client.NewConsole()

	input := `capture C:\Windows\Temp\`
	if !client.App.Shell().AcceptMultiline([]rune(input)) {
		t.Fatalf("AcceptMultiline(%q) = false, want true", input)
	}

	var got []string
	root := &cobra.Command{Use: "root"}
	root.AddCommand(&cobra.Command{
		Use: "capture",
		Run: func(_ *cobra.Command, args []string) {
			got = args
		},
	})
	menu := client.App.ActiveMenu()
	menu.Command = root

	if err := menu.RunCommandLine(context.Background(), input); err != nil {
		t.Fatalf("RunCommandLine(%q) error = %v", input, err)
	}

	want := []string{`C:\Windows\Temp\`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RunCommandLine(%q) args = %q, want %q", input, got, want)
	}
}

func TestNewConsoleJoinsMultilineInput(t *testing.T) {
	client := &Console{}
	client.NewConsole()

	input := "capture alpha\n\tbeta"
	if !client.App.Shell().AcceptMultiline([]rune(input)) {
		t.Fatalf("AcceptMultiline(%q) = false, want true", input)
	}

	var got []string
	root := &cobra.Command{Use: "root"}
	root.AddCommand(&cobra.Command{
		Use: "capture",
		Run: func(_ *cobra.Command, args []string) {
			got = args
		},
	})
	menu := client.App.ActiveMenu()
	menu.Command = root

	if err := menu.RunCommandLine(context.Background(), input); err != nil {
		t.Fatalf("RunCommandLine(%q) error = %v", input, err)
	}

	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RunCommandLine(%q) args = %q, want %q", input, got, want)
	}
}
