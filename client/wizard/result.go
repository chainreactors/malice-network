package wizard

import (
	"strconv"
)

// WizardResult stores the results from a wizard run
type WizardResult struct {
	WizardID string
	Values   map[string]interface{}
}

// NewWizardResult creates a new result instance
func NewWizardResult(wizardID string) *WizardResult {
	return &WizardResult{
		WizardID: wizardID,
		Values:   make(map[string]interface{}),
	}
}

// Set sets a value in the result
func (r *WizardResult) Set(name string, value interface{}) {
	r.Values[name] = value
}

// Get gets a raw value from the result
func (r *WizardResult) Get(name string) interface{} {
	return r.Values[name]
}

// GetString gets a string value from the result
func (r *WizardResult) GetString(name string) string {
	if v, ok := r.Values[name]; ok {
		switch val := v.(type) {
		case string:
			return val
		case *string:
			if val != nil {
				return *val
			}
		}
	}
	return ""
}

// GetBool gets a boolean value from the result
func (r *WizardResult) GetBool(name string) bool {
	if v, ok := r.Values[name]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case *bool:
			if val != nil {
				return *val
			}
		}
	}
	return false
}

// GetInt gets an integer value from the result
func (r *WizardResult) GetInt(name string) int {
	if v, ok := r.Values[name]; ok {
		switch val := v.(type) {
		case int:
			return val
		case *int:
			if val != nil {
				return *val
			}
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		case *string:
			if val != nil {
				if i, err := strconv.Atoi(*val); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

// GetStrings gets a string slice from the result
func (r *WizardResult) GetStrings(name string) []string {
	if v, ok := r.Values[name]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case *[]string:
			if val != nil {
				return *val
			}
		}
	}
	return nil
}

// ToMap returns all values as a map
func (r *WizardResult) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range r.Values {
		// Dereference pointers
		switch val := v.(type) {
		case *string:
			if val != nil {
				result[k] = *val
			} else {
				result[k] = ""
			}
		case *bool:
			if val != nil {
				result[k] = *val
			} else {
				result[k] = false
			}
		case *int:
			if val != nil {
				result[k] = *val
			} else {
				result[k] = 0
			}
		case *[]string:
			if val != nil {
				result[k] = *val
			} else {
				result[k] = []string{}
			}
		default:
			result[k] = v
		}
	}
	return result
}
