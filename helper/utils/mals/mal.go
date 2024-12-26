package mals

import (
	"fmt"
	"strings"
)

func PackMalBinary(data string) string {
	return fmt.Sprintf(`bin:%s`, []byte(data))
}

func UnPackMalBinary(data string) ([]byte, error) {
	parts := strings.SplitN(data, ":", 2)

	if len(parts) != 2 || parts[0] != "bin" {
		return nil, fmt.Errorf("UnPackMalBinary error: invalid binary data format")
	}
	return []byte(parts[1]), nil
}
