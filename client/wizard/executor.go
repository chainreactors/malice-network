package wizard

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Common validators for wizard fields

// ValidateRequired returns a validator that checks if value is non-empty
func ValidateRequired(fieldName string) func(string) error {
	return func(val string) error {
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
		return nil
	}
}

// ValidatePort returns a validator that checks if value is a valid port number
func ValidatePort() func(string) error {
	return func(val string) error {
		if val == "" {
			return nil // Allow empty for optional fields
		}
		port, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid port number: %s", val)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535, got %d", port)
		}
		return nil
	}
}

// ValidateHost returns a validator that checks if value is a valid host (IP or hostname)
// This is more permissive than ValidateIP and allows:
// - IPv4 addresses (e.g., "192.168.1.1", "0.0.0.0")
// - IPv6 addresses (e.g., "::1", "fe80::1")
// - Hostnames (e.g., "localhost", "example.com")
func ValidateHost() func(string) error {
	return func(val string) error {
		if val == "" {
			return nil // Allow empty for optional fields
		}
		// Try parsing as IP first
		if ip := net.ParseIP(val); ip != nil {
			return nil
		}
		// Otherwise accept any non-empty string as hostname
		// (actual DNS resolution happens at connection time)
		return nil
	}
}

// ValidateIP returns a validator that checks if value is a valid IP address (IPv4 or IPv6)
func ValidateIP() func(string) error {
	return func(val string) error {
		if val == "" {
			return nil // Allow empty for optional fields
		}
		if ip := net.ParseIP(val); ip == nil {
			return fmt.Errorf("invalid IP address: %s", val)
		}
		return nil
	}
}

// ValidateRange returns a validator that checks if numeric value is in range
func ValidateRange(min, max int) func(string) error {
	return func(val string) error {
		if val == "" {
			return nil // Allow empty for optional fields
		}
		num, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid number: %s", val)
		}
		if num < min || num > max {
			return fmt.Errorf("value must be between %d and %d, got %d", min, max, num)
		}
		return nil
	}
}

// ValidateFloat returns a validator that checks if value is a valid float in range
func ValidateFloat(min, max float64) func(string) error {
	return func(val string) error {
		if val == "" {
			return nil
		}
		num, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("invalid number: %s", val)
		}
		if num < min || num > max {
			return fmt.Errorf("value must be between %.2f and %.2f, got %.2f", min, max, num)
		}
		return nil
	}
}

// ValidateOneOf returns a validator that checks if value is one of allowed values
func ValidateOneOf(allowed []string) func(string) error {
	return func(val string) error {
		if val == "" {
			return nil
		}
		for _, a := range allowed {
			if val == a {
				return nil
			}
		}
		return fmt.Errorf("value must be one of: %s", strings.Join(allowed, ", "))
	}
}

// CombineValidators combines multiple validators into one
func CombineValidators(validators ...func(string) error) func(string) error {
	return func(val string) error {
		for _, v := range validators {
			if v == nil {
				continue
			}
			if err := v(val); err != nil {
				return err
			}
		}
		return nil
	}
}
