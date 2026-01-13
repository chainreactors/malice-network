package readline

import (
	"fmt"
	"strings"
	"time"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/history"
	"github.com/reeflective/readline/internal/keymap"
)

func (rl *Shell) completionCommands() commands {
	return map[string]func(){
		"complete":               rl.completeWord,
		"possible-completions":   rl.possibleCompletions,
		"insert-completions":     rl.insertCompletions,
		"menu-complete":          rl.menuComplete,
		"menu-complete-backward": rl.menuCompleteBackward,
		"delete-char-or-list":    rl.deleteCharOrList,

		"menu-complete-next-tag":   rl.menuCompleteNextTag,
		"menu-complete-prev-tag":   rl.menuCompletePrevTag,
		"accept-and-menu-complete": rl.acceptAndMenuComplete,
		"vi-registers-complete":    rl.viRegistersComplete,
		"menu-incremental-search":  rl.menuIncrementalSearch,
		"ai-complete":              rl.aiGenerateCommand, // AI command completion (Alt+Q)
	}
}

//
// Commands ---------------------------------------------------------------------------
//

// Attempt completion on the current word.
// Currently identitical to menu-complete.
func (rl *Shell) completeWord() {
	rl.History.SkipSave()

	// This completion function should attempt to insert the first
	// valid completion found, without printing the actual list.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)

		if rl.Config.GetBool("menu-complete-display-prefix") {
			return
		}
	}

	rl.completer.Select(1, 0)
	rl.completer.SkipDisplay()
}

// List possible completions for the current word.
func (rl *Shell) possibleCompletions() {
	rl.History.SkipSave()

	rl.startMenuComplete(rl.commandCompletion)
}

// Insert all completions for the current word into the line.
func (rl *Shell) insertCompletions() {
	rl.History.Save()

	// Generate all possible completions
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)
	}

	// Insert each match, cancel insertion with preserving
	// the candidate just inserted in the line, for each.
	for i := 0; i < rl.completer.Matches(); i++ {
		rl.completer.Select(1, 0)
		rl.completer.Cancel(false, false)
	}

	// Clear the completion menu.
	rl.completer.Cancel(false, false)
	rl.completer.ClearMenu(true)
}

// Like complete-word, except that menu completion is used.
func (rl *Shell) menuComplete() {
	rl.History.SkipSave()

	// No completions are being printed yet, so simply generate the completions
	// as if we just request them without immediately selecting a candidate.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)

		// Immediately select only if not asked to display first.
		if rl.Config.GetBool("menu-complete-display-prefix") {
			return
		}
	}

	rl.completer.Select(1, 0)
}

// Deletes the character under the cursor if not at the
// beginning or end of the line (like delete-char).
// If at the end of the line, behaves identically to
// possible-completions.
func (rl *Shell) deleteCharOrList() {
	switch {
	case rl.cursor.Pos() < rl.line.Len():
		rl.line.CutRune(rl.cursor.Pos())
	default:
		rl.possibleCompletions()
	}
}

// Identical to menu-complete, but moves backward through the
// list of possible completions, as if menu-complete had been
// given a negative argument.
func (rl *Shell) menuCompleteBackward() {
	rl.History.SkipSave()

	// We don't do anything when not already completing.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)
	}

	rl.completer.Select(-1, 0)
}

// In a menu completion, if there are several tags
// of completions, go to the first result of the next tag.
func (rl *Shell) menuCompleteNextTag() {
	rl.History.SkipSave()

	if !rl.completer.IsActive() {
		return
	}

	rl.completer.SelectTag(true)
}

// In a menu completion, if there are several tags of
// completions, go to the first result of the previous tag.
func (rl *Shell) menuCompletePrevTag() {
	rl.History.SkipSave()

	if !rl.completer.IsActive() {
		return
	}

	rl.completer.SelectTag(false)
}

// In a menu completion, insert the current completion
// into the buffer, and advance to the next possible completion.
func (rl *Shell) acceptAndMenuComplete() {
	rl.History.SkipSave()

	// We don't do anything when not already completing.
	if !rl.completer.IsActive() {
		return
	}

	// Also return if no candidate
	if !rl.completer.IsInserting() {
		return
	}

	// First insert the current candidate.
	rl.completer.Cancel(false, false)

	// And cycle to the next one.
	rl.completer.Select(1, 0)
}

// Open a completion menu (similar to menu-complete) with all currently populated Vim registers.
func (rl *Shell) viRegistersComplete() {
	rl.History.SkipSave()
	rl.startMenuComplete(rl.Buffers.Complete)
}

// In a menu completion (whether a candidate is selected or not), start incremental-search
// (fuzzy search) on the results. Search backward incrementally for a specified string.
// The search is case-insensitive if the search string does not have uppercase letters
// and no numeric argument was given. The string may begin with ‘^’ to anchor the search
// to the beginning of the line. A restricted set of editing functions is available in the
// mini-buffer. Keys are looked up in the special isearch keymap, On each change in the
// mini-buffer, any currently selected candidate is dropped from the line and the menu.
// An interrupt signal, as defined by the stty setting, will stop the search and go back to the original line.
func (rl *Shell) menuIncrementalSearch() {
	rl.History.SkipSave()

	// Always regenerate the list of completions.
	rl.completer.GenerateWith(rl.commandCompletion)
	rl.completer.IsearchStart("completions", false, false)
}

//
// Utilities --------------------------------------------------------------------------
//

// startMenuComplete generates a completion menu with completions
// generated from a given completer, without selecting a candidate.
func (rl *Shell) startMenuComplete(completer completion.Completer) {
	rl.History.SkipSave()

	rl.Keymap.SetLocal(keymap.MenuSelect)
	rl.completer.GenerateWith(completer)

	// Start delayed AI completion trigger (1 second delay)
	rl.startDelayedAICompletion()
}

// commandCompletion generates the completions for commands/args/flags.
func (rl *Shell) commandCompletion() completion.Values {
	if rl.Completer == nil {
		return completion.Values{}
	}

	line, cursor := rl.completer.Line()
	comps := rl.Completer(*line, cursor.Pos())

	return comps.convert()
}

// historyCompletion manages the various completion/isearch modes related
// to history control. It can start the history completions, stop them, cycle
// through sources if more than one, and adjust the completion/isearch behavior.
func (rl *Shell) historyCompletion(forward, filterLine, substring bool) {
	switch {
	case rl.Keymap.Local() == keymap.MenuSelect || rl.Keymap.Local() == keymap.Isearch || rl.completer.AutoCompleting():
		// If we are currently completing the last
		// history source, cancel history completion.
		if rl.History.OnLastSource() {
			rl.History.Cycle(true)
			rl.completer.ResetForce()
			rl.Hint.Reset()

			return
		}

		// Else complete the next history source.
		rl.History.Cycle(true)

		fallthrough

	default:
		// Notify if we don't have history sources at all.
		if rl.History.Current() == nil {
			rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgRed, "No command history source", color.Reset))
			return
		}

		// Generate the completions with specified behavior.
		completer := func() completion.Values {
			maxLines := rl.Display.AvailableHelperLines()
			return history.Complete(rl.History, forward, filterLine, maxLines, rl.completer.IsearchRegex)
		}

		if substring {
			rl.completer.GenerateWith(completer)
			rl.completer.IsearchStart(rl.History.Name(), true, true)
		} else {
			rl.startMenuComplete(completer)
			rl.completer.AutocompleteForce()
		}
	}
}

// aiGenerateCommand triggers AI-powered command generation.
// It takes the current input line as natural language and generates a full command.
func (rl *Shell) aiGenerateCommand() {
	if rl.AIGenerateCommand == nil {
		rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgRed, "AI command generation not configured", color.Reset))
		return
	}

	// Get current input line
	currentLine := string(*rl.line)
	if currentLine == "" {
		rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgYellow, "Enter a description to generate a command", color.Reset))
		return
	}

	// Get recent command history
	var history []string
	histSrc := rl.History.Current()
	if histSrc != nil {
		histLen := histSrc.Len()
		start := 0
		if histLen > 20 {
			start = histLen - 20
		}
		for i := start; i < histLen; i++ {
			if cmd, err := histSrc.GetLine(i); err == nil && cmd != "" {
				history = append(history, cmd)
			}
		}
	}

	// Show loading hint
	rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s", color.Dim, "Generating...", color.Reset))
	rl.Display.Refresh()

	// Call AI to generate command
	newCommand, err := rl.AIGenerateCommand(currentLine, history)
	if err != nil {
		rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgRed, "AI Error: "+err.Error(), color.Reset))
		return
	}

	if newCommand == "" {
		rl.Hint.SetTemporary(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgYellow, "No command generated", color.Reset))
		return
	}

	// Clear the current line and insert the generated command
	rl.line.Set([]rune(newCommand)...)
	rl.cursor.Set(len(newCommand))

	// Clear the hint
	rl.Hint.Reset()
}

// startDelayedAICompletion starts the AI completion timer
func (rl *Shell) startDelayedAICompletion() {
	if rl.AISmartComplete == nil {
		return
	}

	// Snapshot line + history on the main loop goroutine.
	currentLine := string(*rl.line)
	if currentLine == "" {
		return
	}

	var history []string
	histSrc := rl.History.Current()
	if histSrc != nil {
		histLen := histSrc.Len()
		start := 0
		if histLen > 20 {
			start = histLen - 20
		}
		for i := start; i < histLen; i++ {
			if cmd, err := histSrc.GetLine(i); err == nil && cmd != "" {
				history = append(history, cmd)
			}
		}
	}

	rl.aiCompletionMu.Lock()
	defer rl.aiCompletionMu.Unlock()

	// Cancel any existing timer
	if rl.aiCompletionTimer != nil {
		rl.aiCompletionTimer.Stop()
		rl.aiCompletionTimer = nil
	}

	// Don't start if already active
	if rl.aiCompletionActive {
		return
	}

	// Start timer for delayed AI completion (1 second)
	rl.aiCompletionTimer = time.AfterFunc(1*time.Second, func() {
		rl.triggerAICompletion(currentLine, history)
	})
}

// triggerAICompletion triggers AI-powered completion asynchronously
func (rl *Shell) triggerAICompletion(currentLine string, history []string) {
	rl.aiCompletionMu.Lock()
	if rl.aiCompletionActive {
		rl.aiCompletionMu.Unlock()
		return
	}
	rl.aiCompletionActive = true
	rl.aiCompletionMu.Unlock()

	defer func() {
		rl.aiCompletionMu.Lock()
		rl.aiCompletionActive = false
		rl.aiCompletionMu.Unlock()
	}()

	currentLine = strings.TrimSpace(currentLine)
	if currentLine == "" {
		return
	}

	showLoading := rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading()
	if showLoading {
		// Show AI loading hint
		rl.Hint.SetTemporary(fmt.Sprintf("%s[AI] Loading...%s", color.FgCyan, color.Reset))
		rl.Display.RefreshHelpers()
	}

	// Call AI completion
	suggestions, err := rl.AISmartComplete(currentLine, history)
	if err != nil {
		// Silently fail - keep showing local completions
		if showLoading && rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
			rl.Hint.Reset()
			rl.Display.RefreshHelpers()
		}
		return
	}

	if len(suggestions) == 0 {
		if showLoading && rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
			rl.Hint.Reset()
			rl.Display.RefreshHelpers()
		}
		return
	}

	// Notify callback if set
	if rl.aiCompletionCallback != nil {
		rl.aiCompletionCallback(suggestions)
	}

	// Queue AI suggestions to be applied by the main loop, or immediately if idle.
	rl.aiCompletionMu.Lock()
	rl.aiCompletionLine = currentLine
	rl.aiCompletionPending = append([]string(nil), suggestions...)
	rl.aiCompletionMu.Unlock()

	if rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
		rl.applyPendingAICompletion(true)
	}
}

// cancelAICompletion cancels any pending AI completion
func (rl *Shell) cancelAICompletion() {
	rl.aiCompletionMu.Lock()
	defer rl.aiCompletionMu.Unlock()

	if rl.aiCompletionTimer != nil {
		rl.aiCompletionTimer.Stop()
		rl.aiCompletionTimer = nil
	}
}

// SetAICompletionCallback sets a callback for when AI completion results arrive
func (rl *Shell) SetAICompletionCallback(callback func(suggestions []string)) {
	rl.aiCompletionMu.Lock()
	defer rl.aiCompletionMu.Unlock()
	rl.aiCompletionCallback = callback
}

func (rl *Shell) applyPendingAICompletion(refresh bool) bool {
	rl.aiCompletionMu.Lock()
	currentLine := rl.aiCompletionLine
	suggestions := rl.aiCompletionPending
	rl.aiCompletionLine = ""
	rl.aiCompletionPending = nil
	rl.aiCompletionMu.Unlock()

	if len(suggestions) == 0 {
		return false
	}

	// Discard stale suggestions.
	baseLine, _ := rl.completer.BaseLine()
	if strings.TrimSpace(string(*baseLine)) != currentLine {
		return false
	}

	// Only enter completion mode if we're not already in another local keymap.
	local := rl.Keymap.Local()
	if local != "" && local != keymap.MenuSelect {
		return false
	}
	if local == "" {
		rl.Keymap.SetLocal(keymap.MenuSelect)
	}

	// Add AI suggestions to the completion menu as a selectable group.
	rl.AppendAICompletions(suggestions)

	// Show hint with count of AI suggestions.
	hintText := fmt.Sprintf("%s[AI] %d suggestions available (use Tab/Arrow to select)%s", color.FgCyan, len(suggestions), color.Reset)
	rl.Hint.SetTemporary(hintText)
	if refresh {
		rl.Display.RefreshHelpers()
	}

	return true
}
