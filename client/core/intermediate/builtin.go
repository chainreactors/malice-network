package intermediate

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"os"
	"path/filepath"
)

func ReadResourceFile(pluginName, filename string) (string, error) {
	resourcePath := filepath.Join(assets.GetMalsDir(), pluginName, "resources", filename)
	content, err := os.ReadFile(resourcePath)
	if err != nil {
		return "", fmt.Errorf("error reading resource file: %v", err)
	}
	return string(content), nil
}
