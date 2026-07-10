package core

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/spf13/cobra"
)

// CheckCommandCompatibility validates command constraints inherited from the
// command tree against the selected session.
func CheckCommandCompatibility(cmd *cobra.Command, sess *client.Session) error {
	for current := cmd; current != nil; current = current.Parent() {
		if err := checkCommandAnnotations(current, sess); err != nil {
			return err
		}
	}
	return nil
}

func checkCommandAnnotations(cmd *cobra.Command, sess *client.Session) error {
	annotations := cmd.Annotations
	if len(annotations) == 0 {
		return nil
	}

	if sess == nil || sess.Session == nil {
		return fmt.Errorf("command %q is unavailable without a session", cmd.CommandPath())
	}

	if required := annotations["os"]; required != "" {
		actual := ""
		if sess.Os != nil {
			actual = sess.Os.Name
		}
		if !matchesCommandConstraint(required, actual, normalizeOS) {
			return incompatibleCommandError(cmd, "os", required, actual)
		}
	}
	if required := annotations["arch"]; required != "" {
		actual := ""
		if sess.Os != nil {
			actual = sess.Os.Arch
		}
		if !matchesCommandConstraint(required, actual, normalizeArch) {
			return incompatibleCommandError(cmd, "arch", required, actual)
		}
	}
	if required := annotations["implant"]; required != "" &&
		!matchesCommandConstraint(required, sess.Type, normalizeConstraintValue) {
		return incompatibleCommandError(cmd, "implant", required, sess.Type)
	}
	if required := annotations["depend"]; required != "" {
		modules := make(map[string]struct{}, len(sess.Modules))
		for _, module := range sess.Modules {
			modules[normalizeConstraintValue(module)] = struct{}{}
		}
		for _, dependency := range splitCommandConstraint(required) {
			if _, ok := modules[normalizeConstraintValue(dependency)]; !ok {
				return incompatibleCommandError(cmd, "depend", dependency, strings.Join(sess.Modules, ","))
			}
		}
	}
	return nil
}

func matchesCommandConstraint(required, actual string, normalize func(string) string) bool {
	actual = normalize(actual)
	for _, candidate := range splitCommandConstraint(required) {
		if normalize(candidate) == actual {
			return true
		}
	}
	return false
}

func splitCommandConstraint(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func normalizeOS(value string) string {
	switch normalizeConstraintValue(value) {
	case "darwin", "macos", "osx":
		return "mac"
	default:
		return normalizeConstraintValue(value)
	}
}

func normalizeArch(value string) string {
	return consts.FormatArch(normalizeConstraintValue(value))
}

func normalizeConstraintValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func incompatibleCommandError(cmd *cobra.Command, constraint, required, actual string) error {
	if actual == "" {
		actual = "unknown"
	}
	return fmt.Errorf(
		"command %q is unavailable: requires %s %q, session has %q",
		cmd.CommandPath(), constraint, required, actual,
	)
}
