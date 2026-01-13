package wizard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chainreactors/malice-network/helper/intl"
	"gopkg.in/yaml.v3"
)

// WizardSpec is a serializable wizard definition (JSON/YAML) for building reusable templates.
type WizardSpec struct {
	ID          string      `json:"id" yaml:"id"`
	Title       string      `json:"title" yaml:"title"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      []FieldSpec `json:"fields" yaml:"fields"`
}

// FieldSpec is a serializable field definition.
type FieldSpec struct {
	Name        string   `json:"name" yaml:"name"`
	Title       string   `json:"title" yaml:"title"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string   `json:"type" yaml:"type"`
	Default     any      `json:"default,omitempty" yaml:"default,omitempty"`
	Options     []string `json:"options,omitempty" yaml:"options,omitempty"`
	Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
}

// SpecFromMap converts a generic map (e.g. from Lua) into a WizardSpec.
func SpecFromMap(spec map[string]interface{}) (*WizardSpec, error) {
	if spec == nil {
		return nil, errors.New("spec is nil")
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var out WizardSpec
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LoadSpec loads a WizardSpec from a JSON/YAML file.
func LoadSpec(path string) (*WizardSpec, error) {
	data, err := readSpecBytes(path)
	if err != nil {
		return nil, err
	}

	var spec WizardSpec
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, err
		}
	default:
		// Try YAML first for better UX (it is a superset for many configs).
		if err := yaml.Unmarshal(data, &spec); err != nil {
			if err2 := json.Unmarshal(data, &spec); err2 != nil {
				return nil, fmt.Errorf("unsupported spec format (expected .json/.yaml/.yml): %w", err)
			}
		}
	}

	return &spec, nil
}

func readSpecBytes(path string) ([]byte, error) {
	if strings.HasPrefix(path, "embed://") {
		return intl.ReadEmbedResource(path)
	}
	return os.ReadFile(path)
}

// NewWizardFromFile loads a WizardSpec and builds a Wizard instance.
func NewWizardFromFile(path string) (*Wizard, error) {
	spec, err := LoadSpec(path)
	if err != nil {
		return nil, err
	}
	return NewWizardFromSpec(spec)
}

// NewWizardFromSpec builds a Wizard from a WizardSpec.
func NewWizardFromSpec(spec *WizardSpec) (*Wizard, error) {
	if spec == nil {
		return nil, errors.New("spec is nil")
	}
	if strings.TrimSpace(spec.ID) == "" {
		return nil, errors.New("spec.id is required")
	}

	wiz := NewWizard(spec.ID, spec.Title).WithDescription(spec.Description)

	for i, fs := range spec.Fields {
		if strings.TrimSpace(fs.Name) == "" {
			return nil, fmt.Errorf("fields[%d].name is required", i)
		}
		if strings.TrimSpace(fs.Title) == "" {
			return nil, fmt.Errorf("fields[%d].title is required", i)
		}
		ft, err := parseFieldTypeName(fs.Type)
		if err != nil {
			return nil, fmt.Errorf("fields[%d].type: %w", i, err)
		}

		field := &WizardField{
			Name:        fs.Name,
			Title:       fs.Title,
			Description: fs.Description,
			Type:        ft,
			Options:     append([]string(nil), fs.Options...),
			Required:    fs.Required,
		}

		if fs.Default != nil {
			switch ft {
			case FieldConfirm:
				b, err := coerceBool(fs.Default)
				if err != nil {
					return nil, fmt.Errorf("fields[%d].default: %w", i, err)
				}
				field.Default = b
			case FieldNumber:
				n, err := coerceInt(fs.Default)
				if err != nil {
					return nil, fmt.Errorf("fields[%d].default: %w", i, err)
				}
				field.Default = n
			case FieldMultiSelect:
				ss, err := coerceStrings(fs.Default)
				if err != nil {
					return nil, fmt.Errorf("fields[%d].default: %w", i, err)
				}
				field.Default = ss
			default:
				field.Default = fmt.Sprintf("%v", fs.Default)
			}
		}

		if ft == FieldSelect || ft == FieldMultiSelect {
			if len(field.Options) == 0 {
				return nil, fmt.Errorf("fields[%d].options is required for %s", i, fs.Type)
			}
		}

		wiz.AddField(field)
	}

	return wiz, nil
}

// RegisterTemplateFromSpec registers a template backed by a WizardSpec.
func RegisterTemplateFromSpec(name string, spec *WizardSpec) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("template name is required")
	}
	if spec == nil {
		return errors.New("spec is nil")
	}

	specCopy := *spec
	if strings.TrimSpace(specCopy.ID) == "" {
		specCopy.ID = name
	}

	wiz, err := NewWizardFromSpec(&specCopy)
	if err != nil {
		return err
	}

	RegisterTemplate(name, func() *Wizard { return wiz.Clone() })
	return nil
}

func parseFieldTypeName(name string) (FieldType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "input":
		return FieldInput, nil
	case "text":
		return FieldText, nil
	case "select":
		return FieldSelect, nil
	case "multiselect", "multi_select", "multi-select":
		return FieldMultiSelect, nil
	case "confirm":
		return FieldConfirm, nil
	case "number", "int", "integer":
		return FieldNumber, nil
	case "filepath", "file_path", "file-path":
		return FieldFilePath, nil
	default:
		return 0, fmt.Errorf("unknown field type: %q", name)
	}
}

func coerceBool(v any) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "1", "true", "yes", "y", "on":
			return true, nil
		case "0", "false", "no", "n", "off":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool: %q", val)
		}
	default:
		return false, fmt.Errorf("invalid bool type: %T", v)
	}
}

func coerceInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return 0, fmt.Errorf("invalid int: %q", val)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("invalid int type: %T", v)
	}
}

func coerceStrings(v any) ([]string, error) {
	switch val := v.(type) {
	case []string:
		out := make([]string, len(val))
		copy(out, val)
		return out, nil
	case []interface{}:
		out := make([]string, 0, len(val))
		for i, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("invalid string at index %d: %T", i, item)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid []string type: %T", v)
	}
}
