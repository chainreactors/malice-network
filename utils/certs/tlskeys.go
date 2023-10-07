package certs

import (
	"fmt"
	"os"
)

var (
	// TLSKeyLogger - File descriptor for logging TLS keys
	TLSKeyLogger = newKeyLogger()
)

func newKeyLogger() *os.File {
	keyFilePath, present := os.LookupEnv("SSLKEYLOGFILE")
	if present {
		keyFile, err := os.OpenFile(keyFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			// TODO - log failed to open TLS key file
			//certsLog.Errorf(fmt.Sprintf("Failed to open TLS key file %v", err))
			return nil
		}
		fmt.Printf("NOTICE: TLS Keys logged to '%s'\n", keyFilePath)
		return keyFile
	}
	return nil
}
