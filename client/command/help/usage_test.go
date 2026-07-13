package help

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestUsageSectionsDoNotRunIntoTheirContent(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "demo [name]",
		Example: "demo example",
		Run:     func(*cobra.Command, []string) {},
	}
	cmd.Flags().String("mode", "", "demo mode")
	var output bytes.Buffer
	cmd.SetOut(&output)

	if err := UsageFunc(cmd); err != nil {
		t.Fatalf("UsageFunc failed: %v", err)
	}
	got := output.String()
	for _, joined := range []string{"## Examples:demo", "## Flags:•", "## Flags:*"} {
		if strings.Contains(got, joined) {
			t.Fatalf("usage section is joined with its content at %q:\n%s", joined, got)
		}
	}
}

func TestFlagUsagesHidesInternalFlagsAndZeroDefaults(t *testing.T) {
	flags := pflag.NewFlagSet("demo", pflag.ContinueOnError)
	flags.String("name", "", "resource name")
	flags.Bool("disable", false, "disable resource")
	flags.String("format", "raw", "output format")
	flags.String("literal-zero", "0", "literal string value")
	flags.String("internal", "", "internal flag")
	if err := flags.MarkHidden("internal"); err != nil {
		t.Fatalf("MarkHidden failed: %v", err)
	}

	got := FlagUsages(flags)
	for _, unwanted := range []string{"--internal", "default: ``", "default: `false`"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("FlagUsages contains %q:\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "default: `raw`") {
		t.Fatalf("FlagUsages omitted meaningful default:\n%s", got)
	}
	if !strings.Contains(got, "default: `0`") {
		t.Fatalf("FlagUsages omitted meaningful string default:\n%s", got)
	}
}
