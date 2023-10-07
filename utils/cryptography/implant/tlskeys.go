package cryptography

import (
	"os"
)

var (
	// TLSKeyLogger - File descriptor for logging TLS keys
	TLSKeyLogger = newKeyLogger()
)

func newKeyLogger() *os.File {
	// {{if .Config.Debug}}
	keyFilePath, present := os.LookupEnv("SSLKEYLOGFILE")
	if present {
		keyFile, err := os.OpenFile(keyFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil
		}
		return keyFile
	}
	// {{end}}
	return nil
}
