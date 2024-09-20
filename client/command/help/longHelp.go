package help

import (
	"bytes"
	_ "embed"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"regexp"
	"strings"
	"text/template"
)

// FormatLongHelp
func FormatLongHelp(long string) string {
	content := removeImages(long)

	return renderMarkdown(content)
}

// renderMarkdown
func renderMarkdown(markdownContent string) string {
	h3Style := lipgloss.NewStyle().Bold(true).MarginBottom(1).MarginTop(1)
	h4Style := lipgloss.NewStyle().Bold(true).MarginBottom(0).MarginTop(1)
	strongStyle := lipgloss.NewStyle().Bold(true)
	liStyle := lipgloss.NewStyle().MarginLeft(2)

	markdownContent = replaceMarkdownHeading(markdownContent, `### (.*)`, h3Style.Render)

	markdownContent = replaceMarkdownHeading(markdownContent, `#### (.*)`, h4Style.Render)

	markdownContent = replaceMarkdownBold(markdownContent, `\*\*(.*?)\*\*`, strongStyle.Render)

	markdownContent = replaceMarkdownInlineCode(markdownContent, "`(.*?)`", "`")

	markdownContent = strings.ReplaceAll(markdownContent, "- ", liStyle.Render("- "))

	return strings.TrimSpace(markdownContent)
}

// replaceMarkdownHeading
func replaceMarkdownHeading(content, pattern string, replacer func(...string) string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		content := re.FindStringSubmatch(match)[1]
		return replacer(content)
	})
}

// replaceMarkdownBold
func replaceMarkdownBold(content, pattern string, replacer func(...string) string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		content := re.FindStringSubmatch(match)[1]
		return replacer(content)
	})
}

// replaceMarkdownInlineCode
func replaceMarkdownInlineCode(content, pattern, delimiter string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		code := re.FindStringSubmatch(match)[1]
		return delimiter + code + delimiter
	})
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
		Black:     termenv.String("").Foreground(tui.Black).String(),
		Red:       termenv.String("").Foreground(tui.Red).String(),
		Green:     termenv.String("").Foreground(tui.Green).String(),
		Orange:    termenv.String("").Foreground(tui.Orange).String(),
		Blue:      termenv.String("").Foreground(tui.Blue).String(),
		Purple:    termenv.String("").Foreground(tui.Purple).String(),
		Cyan:      termenv.String("").Foreground(tui.Cyan).String(),
		Gray:      termenv.String("").Foreground(tui.Gray).String(),
	})
	return outputBuf.String()
}
