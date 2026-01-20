package wizard

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ============ Dynamic Providers ============

// OptionProvider returns dynamic options for a flag's select menu
type OptionProvider func() []string

// DefaultProvider returns a dynamic default value for a flag
type DefaultProvider func() string

var (
	optionProviders   = make(map[string]OptionProvider)
	optionProvidersMu sync.RWMutex

	defaultProviders   = make(map[string]DefaultProvider)
	defaultProvidersMu sync.RWMutex
)

// RegisterProvider registers a dynamic option provider for a flag name
func RegisterProvider(flagName string, fn OptionProvider) {
	optionProvidersMu.Lock()
	defer optionProvidersMu.Unlock()
	optionProviders[flagName] = fn
}

// RegisterDefaultProvider registers a default value provider for a flag name
func RegisterDefaultProvider(flagName string, fn DefaultProvider) {
	defaultProvidersMu.Lock()
	defer defaultProvidersMu.Unlock()
	defaultProviders[flagName] = fn
}

func getOptionProvider(flagName string) (OptionProvider, bool) {
	optionProvidersMu.RLock()
	defer optionProvidersMu.RUnlock()
	fn, ok := optionProviders[flagName]
	return fn, ok
}

func getDefaultProvider(flagName string) (DefaultProvider, bool) {
	defaultProvidersMu.RLock()
	defer defaultProvidersMu.RUnlock()
	fn, ok := defaultProviders[flagName]
	return fn, ok
}

// RunWizard runs an interactive wizard for the given command's flags.
// It returns the collected values as a map, or an error if cancelled.
func RunWizard(cmd *cobra.Command) (map[string]any, error) {
	result := make(map[string]any)
	groups := buildFormGroups(cmd, result)

	if len(groups) == 0 {
		return result, nil
	}

	form := NewGroupedWizardForm(groups)
	if err := form.Run(); err != nil {
		return nil, err
	}

	// Finalize number fields (convert string -> int)
	finalizeResult(result, cmd)

	// Apply result back to flags
	if err := ApplyResultToFlags(cmd, result); err != nil {
		return nil, err
	}

	return result, nil
}

// buildFormGroups creates FormGroups from command flags
func buildFormGroups(cmd *cobra.Command, result map[string]any) []*FormGroup {
	// Collect flags by group
	groups := make(map[string][]*pflag.Flag)
	var ungrouped []*pflag.Flag
	groupOrder := make([]string, 0)
	seen := make(map[string]bool)

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if skipFlag(flag) {
			return
		}
		if g := getFlagGroup(flag); g != "" {
			groups[g] = append(groups[g], flag)
			if !seen[g] {
				groupOrder = append(groupOrder, g)
				seen[g] = true
			}
		} else {
			ungrouped = append(ungrouped, flag)
		}
	})

	// Sort groups by order annotation
	sort.SliceStable(groupOrder, func(i, j int) bool {
		return getGroupOrder(groups[groupOrder[i]]) < getGroupOrder(groups[groupOrder[j]])
	})

	var formGroups []*FormGroup

	// Add ungrouped flags as "General" group
	if len(ungrouped) > 0 {
		sortByOrder(ungrouped)
		formGroups = append(formGroups, &FormGroup{
			Name:   "general",
			Title:  "General",
			Fields: flagsToFields(ungrouped, result),
		})
	}

	// Add grouped flags
	for _, name := range groupOrder {
		flags := groups[name]
		sortByOrder(flags)
		formGroups = append(formGroups, &FormGroup{
			Name:   sanitize(name),
			Title:  name,
			Fields: flagsToFields(flags, result),
		})
	}

	return formGroups
}

// flagsToFields converts a slice of flags to FormFields
func flagsToFields(flags []*pflag.Flag, result map[string]any) []*FormField {
	fields := make([]*FormField, 0, len(flags))
	for _, flag := range flags {
		fields = append(fields, flagToField(flag, result))
	}
	return fields
}

// flagToField converts a single flag to a FormField
func flagToField(flag *pflag.Flag, result map[string]any) *FormField {
	field := &FormField{
		Name:        flag.Name,
		Title:       flag.Name,
		Description: flag.Usage,
		Required:    isRequired(flag),
	}

	// Get current/default value
	val := flag.Value.String()
	if v, ok := getDefaultFromAnnotation(flag); ok {
		val = v
	}

	// Determine field type
	switch flag.Value.Type() {
	case "bool":
		field.Kind = KindConfirm
		field.ConfirmVal = val == "true"
		result[flag.Name] = &field.ConfirmVal
		field.Value = &field.ConfirmVal

	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		field.Kind = KindNumber
		field.InputValue = val
		field.Validate = intValidator(flag.Value.Type())
		result[flag.Name] = &field.InputValue
		field.Value = &field.InputValue

	case "float32", "float64":
		field.Kind = KindInput
		field.InputValue = val
		field.Validate = floatValidator(flag)
		result[flag.Name] = &field.InputValue
		field.Value = &field.InputValue

	default:
		// Check for slice type
		if sv, ok := flag.Value.(pflag.SliceValue); ok {
			field.Kind = KindInput
			field.InputValue = formatCSV(sv.GetSlice())
			field.Description = flag.Usage + " (comma-separated)"
			result[flag.Name] = &field.InputValue
			field.Value = &field.InputValue
			break
		}

		field.Kind = KindInput
		field.InputValue = val
		result[flag.Name] = &field.InputValue
		field.Value = &field.InputValue

		// Check for textarea widget
		if getWidget(flag) == "textarea" {
			// Still KindInput, just noted
		}
	}

	// Check for enum options -> convert to Select
	if opts := getOptions(flag); len(opts) > 0 {
		field.Kind = KindSelect
		field.Options = opts

		// Find selected index
		selected := 0
		found := false
		for i, opt := range opts {
			if opt == val {
				selected = i
				found = true
				break
			}
		}
		// Preserve empty defaults if the options include an empty placeholder.
		if !found && (val == "" || val == "(empty)") {
			for i, opt := range opts {
				if opt == "" || opt == "(empty)" {
					selected = i
					found = true
					break
				}
			}
		}
		// If empty and no empty option exists, select first non-empty.
		if !found && (val == "" || val == "(empty)") {
			for i, opt := range opts {
				if opt != "" && opt != "(empty)" {
					selected = i
					break
				}
			}
		}
		field.Selected = selected

		// Store as string pointer
		strVal := opts[selected]
		result[flag.Name] = &strVal
		field.Value = &strVal
	}

	return field
}

// ApplyResultToFlags applies wizard results back to command flags
func ApplyResultToFlags(cmd *cobra.Command, result map[string]any) error {
	for name, value := range result {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			flag = cmd.PersistentFlags().Lookup(name)
		}
		if flag == nil {
			continue
		}

		strVal := toString(value)
		currentVal := flag.Value.String()

		// Handle slice flags specially
		if sv, ok := flag.Value.(pflag.SliceValue); ok {
			desired, err := parseCSV(strVal)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %w", name, err)
			}
			if !sliceEqual(sv.GetSlice(), desired) {
				if err := sv.Replace(desired); err != nil {
					return fmt.Errorf("failed to set %s: %w", name, err)
				}
				flag.Changed = true
			}
			continue
		}

		// Skip if value unchanged
		if currentVal == strVal {
			continue
		}

		if err := flag.Value.Set(strVal); err != nil {
			return fmt.Errorf("failed to set %s: %w", name, err)
		}
		flag.Changed = true
	}
	return nil
}

// finalizeResult converts number string values to int
func finalizeResult(result map[string]any, cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		switch flag.Value.Type() {
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			if ptr, ok := result[flag.Name].(*string); ok && ptr != nil {
				s := strings.TrimSpace(*ptr)
				if s == "" {
					return
				}
				if parsed, ok := parseNumber(flag.Value.Type(), s); ok {
					result[flag.Name] = parsed
				}
			}
		}
	})
}

// ============ Helpers ============

var skipFlags = map[string]bool{"help": true, "wizard": true, "version": true}

func skipFlag(flag *pflag.Flag) bool {
	return skipFlags[flag.Name] || flag.Hidden
}

func getFlagGroup(flag *pflag.Flag) string {
	if flag.Annotations == nil {
		return ""
	}
	if g, ok := flag.Annotations["ui:group"]; ok && len(g) > 0 {
		return g[0]
	}
	if g, ok := flag.Annotations["group"]; ok && len(g) > 0 {
		return g[0]
	}
	return ""
}

func getGroupOrder(flags []*pflag.Flag) int {
	min := 9999
	for _, f := range flags {
		if o := getFlagOrder(f); o < min {
			min = o
		}
	}
	return min
}

func getFlagOrder(flag *pflag.Flag) int {
	if flag.Annotations == nil {
		return 9999
	}
	if o, ok := flag.Annotations["ui:order"]; ok && len(o) > 0 {
		if n, err := strconv.Atoi(o[0]); err == nil {
			return n
		}
	}
	return 9999
}

func sortByOrder(flags []*pflag.Flag) {
	sort.SliceStable(flags, func(i, j int) bool {
		return getFlagOrder(flags[i]) < getFlagOrder(flags[j])
	})
}

func sanitize(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func isRequired(flag *pflag.Flag) bool {
	if flag.Annotations == nil {
		return false
	}
	if r, ok := flag.Annotations["ui:required"]; ok && len(r) > 0 {
		return r[0] == "true"
	}
	if _, ok := flag.Annotations["cobra_annotation_bash_completion_one_required_flag"]; ok {
		return true
	}
	return false
}

func getDefaultFromAnnotation(flag *pflag.Flag) (string, bool) {
	// 1. Check dynamic provider first
	if provider, ok := getDefaultProvider(flag.Name); ok {
		if val := provider(); val != "" {
			return val, true
		}
	}
	// 2. Check static annotation
	if flag.Annotations != nil {
		if d, ok := flag.Annotations["ui:default"]; ok && len(d) > 0 {
			return d[0], true
		}
	}
	return "", false
}

func getWidget(flag *pflag.Flag) string {
	if flag.Annotations == nil {
		return ""
	}
	if w, ok := flag.Annotations["ui:widget"]; ok && len(w) > 0 {
		return w[0]
	}
	return ""
}

func getOptions(flag *pflag.Flag) []string {
	// 1. Check dynamic provider first
	if provider, ok := getOptionProvider(flag.Name); ok {
		if opts := provider(); len(opts) > 0 {
			return opts
		}
	}
	// 2. Check static annotation
	if flag.Annotations != nil {
		if o, ok := flag.Annotations["ui:options"]; ok && len(o) > 0 {
			return o
		}
	}
	return nil
}

func floatValidator(flag *pflag.Flag) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return fmt.Errorf("invalid number")
		}
		return nil
	}
}

func parseNumber(typeName, s string) (any, bool) {
	switch typeName {
	case "int":
		n, err := strconv.ParseInt(s, 10, strconv.IntSize)
		if err != nil {
			return nil, false
		}
		return int(n), true
	case "int8":
		n, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return nil, false
		}
		return int8(n), true
	case "int16":
		n, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			return nil, false
		}
		return int16(n), true
	case "int32":
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, false
		}
		return int32(n), true
	case "int64":
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, false
		}
		return n, true
	case "uint":
		n, err := strconv.ParseUint(s, 10, strconv.IntSize)
		if err != nil {
			return nil, false
		}
		return uint(n), true
	case "uint8":
		n, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return nil, false
		}
		return uint8(n), true
	case "uint16":
		n, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return nil, false
		}
		return uint16(n), true
	case "uint32":
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, false
		}
		return uint32(n), true
	case "uint64":
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, false
		}
		return n, true
	default:
		return nil, false
	}
}

func intValidator(typeName string) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		var err error
		switch typeName {
		case "int":
			_, err = strconv.ParseInt(s, 10, strconv.IntSize)
		case "int8":
			_, err = strconv.ParseInt(s, 10, 8)
		case "int16":
			_, err = strconv.ParseInt(s, 10, 16)
		case "int32":
			_, err = strconv.ParseInt(s, 10, 32)
		case "int64":
			_, err = strconv.ParseInt(s, 10, 64)
		case "uint":
			_, err = strconv.ParseUint(s, 10, strconv.IntSize)
		case "uint8":
			_, err = strconv.ParseUint(s, 10, 8)
		case "uint16":
			_, err = strconv.ParseUint(s, 10, 16)
		case "uint32":
			_, err = strconv.ParseUint(s, 10, 32)
		case "uint64":
			_, err = strconv.ParseUint(s, 10, 64)
		}
		if err != nil {
			return fmt.Errorf("please enter a valid number")
		}
		return nil
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case *string:
		if val == nil {
			return ""
		}
		return *val
	case *bool:
		if val == nil {
			return "false"
		}
		return strconv.FormatBool(*val)
	case *int:
		if val == nil {
			return "0"
		}
		return strconv.Itoa(*val)
	case int:
		return strconv.Itoa(val)
	case bool:
		return strconv.FormatBool(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatCSV(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.Write(vals)
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}

func parseCSV(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	if s == "" {
		return []string{}, nil
	}
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	return r.Read()
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
