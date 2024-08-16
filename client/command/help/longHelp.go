package help

import (
	_ "embed"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"strings"
)

//go:embed help.md
var helpMarkdown string

// GetHelpFor
func GetHelpFor(commandName string) string {
	content := removeImages(helpMarkdown)

	helpText := extractCommandHelp(content, commandName)
	if helpText != "" {
		return formatForTerminal(helpText)
	}

	return "Help not found."
}

// extractCommandHelp
func extractCommandHelp(htmlContent, commandName string) string {
	commandSectionStart := fmt.Sprintf(`### %s`, commandName)
	startIndex := strings.Index(htmlContent, commandSectionStart)
	if startIndex == -1 {
		return ""
	}

	endIndex := strings.Index(htmlContent[startIndex:], "---")
	if endIndex == -1 {
		endIndex = len(htmlContent)
	} else {
		endIndex += startIndex
	}

	return strings.TrimSpace(htmlContent[startIndex:endIndex])
}

// formatForTerminal
func formatForTerminal(markdownContent string) string {
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
