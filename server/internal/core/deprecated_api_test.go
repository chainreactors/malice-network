package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCodeDoesNotUseDeprecatedSafeGo(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, token := range []string{"SafeGo(", "SafeGoWithInfo(", "SafeGoWithTask("} {
			if strings.Contains(content, token) {
				t.Fatalf("deprecated API %q still used in %s", token, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production tree: %v", err)
	}
}
