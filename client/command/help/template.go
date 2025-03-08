package help

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	"github.com/chainreactors/tui"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var TemplateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"appendIfNotPresent":      appendIfNotPresent,
	"rpad":                    rpad,
	"gt":                      Gt,
	"eq":                      Eq,
	"FlagUsages":              FlagUsages,
	"RenderHelp":              RenderHelp,
	"RenderUsage":             RenderUsage,
	"RenderMarkdown":          renderMarkdownFunc,
	"TrimParentCommand":       trimParentCommand,
}

var initializers []func()
var finalizers []func()

const (
	defaultPrefixMatching   = false
	defaultCommandSorting   = true
	defaultCaseInsensitive  = false
	defaultTraverseRunHooks = false
)

var (
	profile   = termenv.ColorProfile()
	Blue      = profile.Color("#3398DA")
	Yellow    = profile.Color("#F1C40F")
	Purple    = profile.Color("#8D44AD")
	Green     = profile.Color("#2FCB71")
	Red       = profile.Color("#E74C3C")
	Gray      = profile.Color("#BDC3C7")
	DarkGray  = profile.Color("#808080")
	Cyan      = profile.Color("#1ABC9C")
	Orange    = profile.Color("#E67E22")
	Black     = profile.Color("#000000")
	Pink      = profile.Color("#EE82EE")
	SlateBlue = profile.Color("#6A5ACD")
	White     = profile.Color("#FFFFFF")
)

// EnablePrefixMatching allows setting automatic prefix matching. Automatic prefix matching can be a dangerous thing
// to automatically enable in CLI tools.
// Set this to true to enable it.
var EnablePrefixMatching = defaultPrefixMatching

// EnableCommandSorting controls sorting of the slice of commands, which is turned on by default.
// To disable sorting, set it to false.
var EnableCommandSorting = defaultCommandSorting

// EnableCaseInsensitive allows case-insensitive commands names. (case sensitive by default)
var EnableCaseInsensitive = defaultCaseInsensitive

// EnableTraverseRunHooks executes persistent pre-run and post-run hooks from all parents.
// By default this is disabled, which means only the first run hook to be found is executed.
var EnableTraverseRunHooks = defaultTraverseRunHooks

// MousetrapHelpText enables an information splash screen on Windows
// if the CLI is started from explorer.exe.
// To disable the mousetrap, just set this variable to blank string ("").
// Works only on Microsoft Windows.
var MousetrapHelpText = `This is a command line tool.

You need to open cmd.exe and run it from there.
`

// MousetrapDisplayDuration controls how long the MousetrapHelpText message is displayed on Windows
// if the CLI is started from explorer.exe. Set to 0 to wait for the return key to be pressed.
// To disable the mousetrap, just set MousetrapHelpText to blank string ("").
// Works only on Microsoft Windows.
var MousetrapDisplayDuration = 5 * time.Second

// AddTemplateFunc adds a template function that's available to Usage and Help
// template generation.
func AddTemplateFunc(name string, tmplFunc interface{}) {
	TemplateFuncs[name] = tmplFunc
}

// AddTemplateFuncs adds multiple template functions that are available to Usage and
// Help template generation.
func AddTemplateFuncs(tmplFuncs template.FuncMap) {
	for k, v := range tmplFuncs {
		TemplateFuncs[k] = v
	}
}

// OnInitialize sets the passed functions to be run when each command's
// Execute method is called.
func OnInitialize(y ...func()) {
	initializers = append(initializers, y...)
}

// OnFinalize sets the passed functions to be run when each command's
// Execute method is terminated.
func OnFinalize(y ...func()) {
	finalizers = append(finalizers, y...)
}

// FIXME Gt is unused by cobra and should be removed in a version 2. It exists only for compatibility with users of cobra.

// Gt takes two types and checks whether the first type is greater than the second. In case of types Arrays, Chans,
// Maps and Slices, Gt will compare their lengths. Ints are compared directly while strings are first parsed as
// ints and then compared.
func Gt(a interface{}, b interface{}) bool {
	var left, right int64
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = int64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = av.Int()
	case reflect.String:
		left, _ = strconv.ParseInt(av.String(), 10, 64)
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = int64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = bv.Int()
	case reflect.String:
		right, _ = strconv.ParseInt(bv.String(), 10, 64)
	}

	return left > right
}

// FIXME Eq is unused by cobra and should be removed in a version 2. It exists only for compatibility with users of cobra.

// Eq takes two types and checks whether they are equal. Supported types are int and string. Unsupported types will panic.
func Eq(a interface{}, b interface{}) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		panic("Eq called on unsupported type")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() == bv.Int()
	case reflect.String:
		return av.String() == bv.String()
	}
	return false
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// FIXME appendIfNotPresent is unused by cobra and should be removed in a version 2. It exists only for compatibility with users of cobra.

// appendIfNotPresent will append stringToAppend to the end of s, but only if it's not yet present in s.
func appendIfNotPresent(s, stringToAppend string) string {
	if strings.Contains(s, stringToAppend) {
		return s
	}
	return s + " " + stringToAppend
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	formattedString := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(formattedString, s)
}

//// tmpl executes the given template text on data, writing the result to w.
//func tmpl(w io.Writer, text string, data interface{}) error {
//	t := template.New("top")
//	t.Funcs(TemplateFuncs)
//	template.Must(t.Parse(text))
//	return t.Execute(w, data)
//}
//
//// ld compares two strings and returns the levenshtein distance between them.
//func ld(s, t string, ignoreCase bool) int {
//	if ignoreCase {
//		s = strings.ToLower(s)
//		t = strings.ToLower(t)
//	}
//	d := make([][]int, len(s)+1)
//	for i := range d {
//		d[i] = make([]int, len(t)+1)
//		d[i][0] = i
//	}
//	for j := range d[0] {
//		d[0][j] = j
//	}
//	for j := 1; j <= len(t); j++ {
//		for i := 1; i <= len(s); i++ {
//			if s[i-1] == t[j-1] {
//				d[i][j] = d[i-1][j-1]
//			} else {
//				min := d[i-1][j]
//				if d[i][j-1] < min {
//					min = d[i][j-1]
//				}
//				if d[i-1][j-1] < min {
//					min = d[i-1][j-1]
//				}
//				d[i][j] = min + 1
//			}
//		}
//
//	}
//	return d[len(s)][len(t)]
//}
//
//func stringInSlice(a string, list []string) bool {
//	for _, b := range list {
//		if b == a {
//			return true
//		}
//	}
//	return false
//}

// CheckErr prints the msg with the prefix 'Error:' and exits with error code 1. If the msg is nil, it does nothing.
func CheckErr(msg interface{}) {
	if msg != nil {
		fmt.Fprintln(os.Stderr, "Error:", msg)
		os.Exit(1)
	}
}

// WriteStringAndCheck writes a string into a buffer, and checks if the error is not nil.
func WriteStringAndCheck(b io.StringWriter, s string) {
	_, err := b.WriteString(s)
	CheckErr(err)
}

// FlagUsages returns a string containing the usage information for all flags in
// the FlagSet. Flags are grouped by their annotations in markdown format.
func FlagUsages(f *pflag.FlagSet) string {
	var s strings.Builder
	groups := make(map[string][]*pflag.Flag)
	var ungroupedFlags []*pflag.Flag

	f.VisitAll(func(flag *pflag.Flag) {
		if group, ok := flag.Annotations["group"]; ok && len(group) > 0 {
			groups[group[0]] = append(groups[group[0]], flag)
		} else {
			ungroupedFlags = append(ungroupedFlags, flag)
		}
	})

	if len(ungroupedFlags) > 0 {
		for _, flag := range ungroupedFlags {
			if flag.Shorthand == "" {
				fmt.Fprintf(&s, "* --%s: %s (default: `%s`)\n", flag.Name, flag.Usage, flag.DefValue)
			} else {
				fmt.Fprintf(&s, "* -%s, --%s: %s (default: `%s`)\n", flag.Shorthand, flag.Name, flag.Usage, flag.DefValue)
			}
		}
		s.WriteString("\n")
	}

	for groupName, flags := range groups {
		if len(flags) > 0 {
			fmt.Fprintf(&s, "### %s\n\n", groupName)
			for _, flag := range flags {
				if flag.Shorthand == "" {
					fmt.Fprintf(&s, "* --%s: %s (default: `%s`)\n", flag.Name, flag.Usage, flag.DefValue)
				} else {
					fmt.Fprintf(&s, "* -%s, --%s: %s (default: `%s`)\n", flag.Shorthand, flag.Name, flag.Usage, flag.DefValue)
				}
			}
			s.WriteString("\n")
		}
	}

	return s.String()
}

func renderMarkdown(markdownContent string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithColorProfile(termenv.ANSI),
		glamour.WithEmoji(),
	)
	if err != nil {
		return markdownContent
	}

	rendered, err := r.Render(strings.TrimSpace(markdownContent))
	if err != nil {
		return markdownContent
	}

	return rendered
}

// removeImages
func removeImages(markdownContent string) string {
	re := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	return re.ReplaceAllString(markdownContent, "")
}

// FormatHelpTmpl - Applies format template to help string
func FormatHelpTmpl(helpStr string) string {
	outputBuf := bytes.NewBufferString("")
	tmpl, _ := template.New("help").Delims("[[", "]]").Parse(helpStr)
	tmpl.Execute(outputBuf, struct {
		Normal    string
		Bold      string
		Underline string
		Black     string
		Red       string
		Green     string
		Orange    string
		Blue      string
		Purple    string
		Cyan      string
		Gray      string
	}{
		Normal:    tui.Normal,
		Bold:      tui.Bold,
		Underline: tui.Underline,
		Black:     termenv.String("").Foreground(Black).String(),
		Red:       termenv.String("").Foreground(Red).String(),
		Green:     termenv.String("").Foreground(Green).String(),
		Orange:    termenv.String("").Foreground(Orange).String(),
		Blue:      termenv.String("").Foreground(Blue).String(),
		Purple:    termenv.String("").Foreground(Purple).String(),
		Cyan:      termenv.String("").Foreground(Cyan).String(),
		Gray:      termenv.String("").Foreground(Gray).String(),
	})
	return outputBuf.String()
}

var (
	renderer     *glamour.TermRenderer
	rendererOnce sync.Once
)

func getMarkdownRenderer() (*glamour.TermRenderer, error) {
	var err error
	rendererOnce.Do(func() {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithColorProfile(termenv.ANSI),
			glamour.WithEmoji(),
		)
	})
	return renderer, err
}

var renderMarkdownFunc = func(title string) string {
	r, err := getMarkdownRenderer()
	if err != nil {
		return strings.TrimSpace(title)
	}

	rendered, err := r.Render(title)
	if err != nil {
		return strings.TrimSpace(title)
	}

	return strings.TrimSpace(rendered)
}

var trimParentCommand = func(useLine string, cmd *cobra.Command) string {
	if cmd.Parent() != nil {
		parentName := cmd.Parent().Name()
		return strings.TrimPrefix(useLine, parentName+" ")
	}
	return useLine
}
