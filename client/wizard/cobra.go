package wizard

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// skipFlags defines flags that should not appear in wizard forms
var skipFlags = map[string]bool{
	"help":    true,
	"wizard":  true,
	"version": true,
}

// CobraToWizard converts a cobra.Command's flags to a Wizard
// It supports grouping via ui:group annotations and edit mode (reading current flag values)
func CobraToWizard(cmd *cobra.Command) *Wizard {
	wiz := NewWizard(wizardIDFromCommand(cmd), cmd.Short)
	if cmd.Long != "" {
		wiz.WithDescription(cmd.Long)
	}

	// Collect all flags and group them
	groups := make(map[string][]*pflag.Flag)
	var ungroupedFlags []*pflag.Flag
	groupOrder := make([]string, 0)
	groupOrderSet := make(map[string]bool)

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if shouldSkipFlag(flag) {
			return
		}

		groupName := getFlagGroup(flag)
		if groupName != "" {
			groups[groupName] = append(groups[groupName], flag)
			if !groupOrderSet[groupName] {
				groupOrder = append(groupOrder, groupName)
				groupOrderSet[groupName] = true
			}
		} else {
			ungroupedFlags = append(ungroupedFlags, flag)
		}
	})

	// Sort groups by order annotation if available
	sort.SliceStable(groupOrder, func(i, j int) bool {
		orderI := getGroupOrder(groupOrder[i], groups[groupOrder[i]])
		orderJ := getGroupOrder(groupOrder[j], groups[groupOrder[j]])
		return orderI < orderJ
	})

	// If there are groups, use grouped mode
	if len(groups) > 0 {
		usedGroupIDs := make(map[string]bool)

		// Add ungrouped fields as "Basic" group first
		if len(ungroupedFlags) > 0 {
			sortFlagsByOrder(ungroupedFlags)
			basicGroup := wiz.NewGroup(uniqueGroupID(usedGroupIDs, "general"), "基础配置")
			for _, flag := range ungroupedFlags {
				field := flagToWizardField(flag)
				basicGroup.AddField(field)
			}
		}

		// Add other groups in order
		for _, groupName := range groupOrder {
			flags := groups[groupName]
			group := wiz.NewGroup(uniqueGroupID(usedGroupIDs, sanitizeGroupName(groupName)), groupName)

			// Sort flags within group by order annotation
			sortFlagsByOrder(flags)

			for _, flag := range flags {
				field := flagToWizardField(flag)
				group.AddField(field)
			}
		}
	} else {
		// Flat mode: all fields without grouping
		sortFlagsByOrder(ungroupedFlags)
		for _, flag := range ungroupedFlags {
			field := flagToWizardField(flag)
			wiz.AddField(field)
		}
	}

	return wiz
}

// shouldSkipFlag determines if a flag should be excluded from wizard
func shouldSkipFlag(flag *pflag.Flag) bool {
	if skipFlags[flag.Name] {
		return true
	}
	// Skip hidden flags
	if flag.Hidden {
		return true
	}
	return false
}

// getFlagGroup extracts the group name from flag annotations
func getFlagGroup(flag *pflag.Flag) string {
	if flag.Annotations == nil {
		return ""
	}
	// Check ui:group annotation
	if groups, ok := flag.Annotations["ui:group"]; ok && len(groups) > 0 {
		return groups[0]
	}
	// Check group annotation (alternative)
	if groups, ok := flag.Annotations["group"]; ok && len(groups) > 0 {
		return groups[0]
	}
	return ""
}

// sanitizeGroupName converts a display name to a valid identifier
func sanitizeGroupName(name string) string {
	// Replace spaces and special chars with underscores
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	return result
}

func uniqueGroupID(used map[string]bool, base string) string {
	idBase := sanitizeGroupName(strings.TrimSpace(base))
	if idBase == "" {
		idBase = "group"
	}
	id := idBase
	if !used[id] {
		used[id] = true
		return id
	}
	for i := 2; ; i++ {
		id = fmt.Sprintf("%s_%d", idBase, i)
		if !used[id] {
			used[id] = true
			return id
		}
	}
}

func wizardIDFromCommand(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := strings.TrimSpace(cmd.CommandPath())
	if path == "" {
		path = cmd.Name()
	}
	// Keep it stable + filesystem-ish.
	id := strings.ToLower(path)
	id = strings.ReplaceAll(id, " ", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, "/", "_")
	return id
}

// getGroupOrder returns the order value for a group (based on first flag's order)
func getGroupOrder(groupName string, flags []*pflag.Flag) int {
	minOrder := 9999
	for _, flag := range flags {
		if order := getFlagOrder(flag); order < minOrder {
			minOrder = order
		}
	}
	return minOrder
}

// getFlagOrder gets the order value from flag annotations
func getFlagOrder(flag *pflag.Flag) int {
	if flag.Annotations == nil {
		return 9999
	}
	if orders, ok := flag.Annotations["ui:order"]; ok && len(orders) > 0 {
		if order, err := strconv.Atoi(orders[0]); err == nil {
			return order
		}
	}
	return 9999
}

// sortFlagsByOrder sorts flags by their ui:order annotation
func sortFlagsByOrder(flags []*pflag.Flag) {
	sort.SliceStable(flags, func(i, j int) bool {
		return getFlagOrder(flags[i]) < getFlagOrder(flags[j])
	})
}

// flagToWizardField converts a single pflag.Flag to WizardField
func flagToWizardField(flag *pflag.Flag) *WizardField {
	field := &WizardField{
		Name:        flag.Name,
		Title:       flag.Name,
		Description: flag.Usage,
		Required:    isFlagRequired(flag),
	}

	// Get current value for edit mode support
	currentValue := flag.Value.String()

	// Determine field type and default value based on flag type
	switch flag.Value.Type() {
	case "bool":
		field.Type = FieldConfirm
		field.Default = currentValue == "true"

	case "int", "int8", "int16", "int32", "int64":
		field.Type = FieldNumber
		if val, err := strconv.ParseInt(currentValue, 10, 64); err == nil {
			field.Default = int(val)
		} else {
			field.Default = 0
		}

	case "uint", "uint8", "uint16", "uint32", "uint64":
		field.Type = FieldNumber
		if val, err := strconv.ParseUint(currentValue, 10, 64); err == nil {
			field.Default = int(val)
		} else {
			field.Default = 0
		}

	case "float32", "float64":
		// Wizard "number" currently supports ints only; treat floats as string input with float validation.
		field.Type = FieldInput
		field.Default = currentValue
		field.Validate = floatValidatorFromFlag(flag)

	default:
		// pflag slice types stringify as "[...]" and Set() may append when already-changed.
		// Treat slices as comma-separated string input and apply via SliceValue.Replace.
		if sv, ok := flag.Value.(pflag.SliceValue); ok {
			field.Type = FieldInput
			field.Default = formatCSV(sv.GetSlice())
			field.Description = flag.Usage + " (逗号分隔多个值)"
			field.Validate = csvListValidator()
			break
		}

		field.Type = FieldInput
		field.Default = currentValue

		// Check if should use textarea
		if widget := getFlagWidget(flag); widget == "textarea" {
			field.Type = FieldText
		}
	}

	// Check for enum options (converts to Select)
	if options := getFlagOptions(flag); len(options) > 0 {
		field.Type = FieldSelect
		field.Options = options
		// Auto-select first non-empty option if current value is empty
		if currentValue == "" || currentValue == "(empty)" {
			for _, opt := range options {
				if opt != "" && opt != "(empty)" {
					field.Default = opt
					break
				}
			}
		} else {
			field.Default = currentValue
		}
	}

	return field
}

// isFlagRequired determines if a flag is required
func isFlagRequired(flag *pflag.Flag) bool {
	if flag.Annotations == nil {
		return false
	}

	// Check ui:required annotation
	if required, ok := flag.Annotations["ui:required"]; ok && len(required) > 0 {
		return required[0] == "true"
	}

	// Check cobra's required flag annotation (set by MarkFlagRequired)
	if required, ok := flag.Annotations["cobra_annotation_bash_completion_one_required_flag"]; ok {
		return len(required) > 0
	}

	return false
}

// getFlagWidget gets the widget type from flag annotations
func getFlagWidget(flag *pflag.Flag) string {
	if flag.Annotations == nil {
		return ""
	}
	if widget, ok := flag.Annotations["ui:widget"]; ok && len(widget) > 0 {
		return widget[0]
	}
	return ""
}

// getFlagOptions gets the options list from flag annotations
func getFlagOptions(flag *pflag.Flag) []string {
	if flag.Annotations == nil {
		return nil
	}
	if options, ok := flag.Annotations["ui:options"]; ok && len(options) > 0 {
		return options
	}
	return nil
}

func csvListValidator() func(string) error {
	return func(s string) error {
		_, err := parseCSVList(s)
		if err != nil {
			return err
		}
		return nil
	}
}

func floatValidatorFromFlag(flag *pflag.Flag) func(string) error {
	min, max, ok := getFlagFloatRange(flag)
	if ok {
		return ValidateFloat(min, max)
	}
	return func(val string) error {
		if strings.TrimSpace(val) == "" {
			return nil
		}
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return fmt.Errorf("invalid number: %s", val)
		}
		return nil
	}
}

func getFlagFloatRange(flag *pflag.Flag) (min, max float64, ok bool) {
	if flag == nil || flag.Annotations == nil {
		return 0, 0, false
	}
	mins, okMin := flag.Annotations["ui:min"]
	maxs, okMax := flag.Annotations["ui:max"]
	if !okMin || !okMax || len(mins) == 0 || len(maxs) == 0 {
		return 0, 0, false
	}
	min, err1 := strconv.ParseFloat(strings.TrimSpace(mins[0]), 64)
	max, err2 := strconv.ParseFloat(strings.TrimSpace(maxs[0]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return min, max, true
}

func formatCSV(vals []string) string {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.Write(vals)
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}

func parseCSVList(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") && len(s) >= 2 {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	if s == "" {
		return []string{}, nil
	}
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	records, err := r.Read()
	if err != nil {
		return nil, err
	}
	return records, nil
}

// ApplyWizardResultToFlags applies wizard results back to cobra.Command flags
func ApplyWizardResultToFlags(cmd *cobra.Command, result *WizardResult) error {
	for name, value := range result.ToMap() {
		// Look up flag in command flags
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			// If flags were not merged (e.g., helper calls outside cobra execution), try persistent flags too.
			flag = cmd.PersistentFlags().Lookup(name)
		}
		if flag == nil {
			// Skip unknown fields
			continue
		}

		changed, err := applyWizardValueToFlag(flag, value)
		if err != nil {
			return fmt.Errorf("failed to set flag %s: %w", name, err)
		}
		if changed {
			flag.Changed = true
		}
	}
	return nil
}

func applyWizardValueToFlag(flag *pflag.Flag, value any) (bool, error) {
	// Slice flags: always replace, never Set() (Set() may append when already-changed).
	if sv, ok := flag.Value.(pflag.SliceValue); ok {
		desired, err := coerceStringSlice(value)
		if err != nil {
			return false, err
		}
		if stringSliceEqual(sv.GetSlice(), desired) {
			return false, nil
		}
		if err := sv.Replace(desired); err != nil {
			return false, err
		}
		return true, nil
	}

	desiredStr := coerceString(value)

	// Best-effort semantic equality to avoid flipping Flag.Changed when user accepted defaults.
	switch flag.Value.Type() {
	case "bool":
		cur, err1 := strconv.ParseBool(flag.Value.String())
		des, err2 := strconv.ParseBool(desiredStr)
		if err1 == nil && err2 == nil && cur == des {
			return false, nil
		}
	case "int", "int8", "int16", "int32", "int64":
		cur, err1 := strconv.ParseInt(flag.Value.String(), 10, 64)
		des, err2 := strconv.ParseInt(strings.TrimSpace(desiredStr), 10, 64)
		if err1 == nil && err2 == nil && cur == des {
			return false, nil
		}
	case "uint", "uint8", "uint16", "uint32", "uint64":
		cur, err1 := strconv.ParseUint(flag.Value.String(), 10, 64)
		des, err2 := strconv.ParseUint(strings.TrimSpace(desiredStr), 10, 64)
		if err1 == nil && err2 == nil && cur == des {
			return false, nil
		}
	case "float32", "float64":
		cur, err1 := strconv.ParseFloat(flag.Value.String(), 64)
		des, err2 := strconv.ParseFloat(strings.TrimSpace(desiredStr), 64)
		if err1 == nil && err2 == nil && cur == des {
			return false, nil
		}
	default:
		if flag.Value.String() == desiredStr {
			return false, nil
		}
	}

	if err := flag.Value.Set(desiredStr); err != nil {
		return false, err
	}
	return true, nil
}

func coerceString(v any) string {
	switch val := v.(type) {
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []string:
		return strings.Join(val, ",")
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func coerceStringSlice(v any) ([]string, error) {
	switch val := v.(type) {
	case []string:
		out := make([]string, len(val))
		copy(out, val)
		return out, nil
	case string:
		return parseCSVList(val)
	default:
		return parseCSVList(coerceString(v))
	}
}

func stringSliceEqual(a, b []string) bool {
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
