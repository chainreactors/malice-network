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
	}
}

//
// Commands ---------------------------------------------------------------------------
//

// Attempt completion on the current word.
func (rl *Shell) completeWord() {
	rl.History.SkipSave()

	// If there's a local suggestion and we're not in completion menu, accept it
	if rl.hasLocalSuggestion() && !rl.completer.IsActive() {
		if rl.acceptLocalSuggestion() {
			return
		}
	}

	// This completion function should attempt to insert the first
	// valid completion found, without printing the actual list.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)

		if rl.Config.GetBool("menu-complete-display-prefix") {
			// Trigger local suggestion for next argument
			rl.startDelayedLocalSuggestion()
			return
		}
	}

	rl.completer.Select(1, 0)
	rl.completer.SkipDisplay()

	// Trigger local suggestion for next argument (after we selected a candidate).
	rl.startDelayedLocalSuggestion()
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

//
// AI Prediction (inline ghost text) -------------------------------------------------------
//

// startDelayedAIPrediction starts the AI prediction timer
func (rl *Shell) startDelayedAIPrediction() {
	if rl.AIPredictNext == nil {
		return
	}

	// Snapshot line + history on the main loop goroutine.
	line, _ := rl.completer.Line()
	if line == nil {
		return
	}
	currentLine := string(*line)
	if currentLine == "" {
		return
	}

	var historyLines []string
	histSrc := rl.History.Current()
	if histSrc != nil {
		histLen := histSrc.Len()
		start := 0
		if histLen > 20 {
			start = histLen - 20
		}
		for i := start; i < histLen; i++ {
			if cmd, err := histSrc.GetLine(i); err == nil && cmd != "" {
				historyLines = append(historyLines, cmd)
			}
		}
	}

	rl.aiPredictionMu.Lock()
	defer rl.aiPredictionMu.Unlock()

	// Cancel any existing timer
	if rl.aiPredictionTimer != nil {
		rl.aiPredictionTimer.Stop()
		rl.aiPredictionTimer = nil
	}

	rl.aiPredictionSeq++
	seq := rl.aiPredictionSeq

	delay := 250 * time.Millisecond
	if strings.HasSuffix(currentLine, " ") || strings.HasSuffix(currentLine, "\t") {
		delay = 80 * time.Millisecond
	}

	if rl.Config != nil {
		if strings.HasSuffix(currentLine, " ") || strings.HasSuffix(currentLine, "\t") {
			if ms := rl.Config.GetInt("ai-prediction-delay-space"); ms > 0 {
				delay = time.Duration(ms) * time.Millisecond
			}
		} else {
			if ms := rl.Config.GetInt("ai-prediction-delay"); ms > 0 {
				delay = time.Duration(ms) * time.Millisecond
			}
		}
	}

	rl.aiPredictionTimer = time.AfterFunc(delay, func() {
		rl.triggerAIPrediction(seq, currentLine, historyLines)
	})
}

// triggerAIPrediction triggers AI-powered prediction asynchronously
func (rl *Shell) triggerAIPrediction(seq uint64, currentLine string, historyLines []string) {
	rl.aiPredictionMu.Lock()
	if seq != rl.aiPredictionSeq {
		rl.aiPredictionMu.Unlock()
		return
	}
	rl.aiPredictionMu.Unlock()

	if currentLine == "" {
		return
	}

	// Drop stale predictions if the line has changed since scheduling.
	line, _ := rl.completer.Line()
	if line == nil || string(*line) != currentLine {
		return
	}

	// Call AI prediction
	prediction, err := rl.AIPredictNext(currentLine, historyLines)
	if err != nil || prediction == "" {
		return
	}

	// Drop stale predictions if the line changed while the request was running.
	line, _ = rl.completer.Line()
	if line == nil || string(*line) != currentLine {
		return
	}

	// Store prediction
	prediction = strings.TrimLeft(prediction, " \t")
	insertText := aiPredictionInsertText(currentLine, prediction)
	if insertText == "" {
		return
	}
	fullSuggestion := currentLine + insertText
	rl.aiPredictionMu.Lock()
	if seq != rl.aiPredictionSeq {
		rl.aiPredictionMu.Unlock()
		return
	}
	rl.aiPrediction = fullSuggestion
	rl.aiPredictionLine = currentLine
	rl.aiPredictionMu.Unlock()

	// Show prediction as inline ghost text (fish-style).
	rl.SetAISuggestion(fullSuggestion)

	// If the main readline loop is currently blocked waiting for input,
	// proactively refresh so the suggestion appears immediately.
	// This avoids waiting for the next keypress.
	if rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
		if rl.Config != nil && !rl.Config.GetBool("ai-prediction-auto-refresh") {
			return
		}

		rl.Display.RefreshLine()
	}
}

// acceptAIPrediction accepts the current AI prediction and inserts it
func (rl *Shell) acceptAIPrediction() bool {
	rl.aiPredictionMu.Lock()
	suggestion := rl.aiPrediction
	rl.aiPredictionMu.Unlock()

	if suggestion == "" {
		return false
	}

	line, _ := rl.completer.Line()
	if line == nil {
		return false
	}
	currentLine := string(*line)
	if currentLine == "" || !strings.HasPrefix(suggestion, currentLine) || len(suggestion) <= len(currentLine) {
		rl.ClearAIPrediction()
		return false
	}

	// We only render the ghost text at end-of-line; keep acceptance consistent.
	if rl.cursor.Pos() != rl.line.Len() {
		return false
	}

	// Accept any virtually inserted completion candidate and exit completion mode.
	completion.UpdateInserted(rl.completer)

	// Insert prediction
	suffix := suggestion[len(currentLine):]
	for _, r := range suffix {
		rl.line.Insert(rl.cursor.Pos(), r)
		rl.cursor.Inc()
	}

	rl.ClearAIPrediction()

	// Refresh display
	rl.Display.Refresh()

	return true
}

// ClearAIPrediction clears any pending AI prediction
func (rl *Shell) ClearAIPrediction() {
	rl.aiPredictionMu.Lock()
	defer rl.aiPredictionMu.Unlock()

	// Invalidate any in-flight predictions/timers.
	rl.aiPredictionSeq++

	if rl.aiPredictionTimer != nil {
		rl.aiPredictionTimer.Stop()
		rl.aiPredictionTimer = nil
	}
	rl.aiPrediction = ""
	rl.aiPredictionLine = ""

	// Clear inline suggestion
	rl.Display.ClearAISuggestion()
}

// GetAIPrediction returns the current AI prediction (for display purposes)
func (rl *Shell) GetAIPrediction() string {
	// Only consider predictions when the cursor is at end-of-line, since the
	// ghost text is rendered there.
	if rl.cursor.Pos() != rl.line.Len() {
		return ""
	}

	rl.aiPredictionMu.Lock()
	suggestion := rl.aiPrediction
	rl.aiPredictionMu.Unlock()
	if suggestion == "" {
		return ""
	}

	line, _ := rl.completer.Line()
	if line == nil {
		return ""
	}
	currentLine := string(*line)
	if currentLine == "" || !strings.HasPrefix(suggestion, currentLine) || len(suggestion) <= len(currentLine) {
		return ""
	}

	return suggestion[len(currentLine):]
}

// aiPredictionInsertText returns the suffix to insert/display for a given
// predicted next argument/value and the current input line.
//
// The predictor is instructed to return a single "next argument/value". In
// practice that can also mean a completion of the current token (e.g. when the
// user already typed a prefix). In that case we should only insert the
// remaining suffix rather than adding a new space-delimited token.
func aiPredictionInsertText(currentLine, prediction string) string {
	prediction = strings.TrimLeft(prediction, " \t")
	if prediction == "" {
		return ""
	}
	if currentLine == "" {
		return prediction
	}

	// If we're already at an argument boundary, insert the prediction as-is.
	lastChar := currentLine[len(currentLine)-1]
	if lastChar == ' ' || lastChar == '\t' {
		return prediction
	}

	// Otherwise, try to interpret the prediction as a completion of the current token.
	tokenStart := strings.LastIndexAny(currentLine, " \t") + 1
	if tokenStart < 0 || tokenStart > len(currentLine) {
		tokenStart = 0
	}

	tokenPrefix := currentLine[tokenStart:]
	if tokenPrefix != "" && strings.HasPrefix(prediction, tokenPrefix) {
		return prediction[len(tokenPrefix):]
	}

	// Fallback: treat as the next token and add a separating space.
	return " " + prediction
}

// applyPendingAICompletion is kept for compatibility with async completion hooks.
// It currently ensures stale AI predictions are discarded when the input line changes.
func (rl *Shell) applyPendingAICompletion(refresh bool) bool {
	// Ensure inline suggestions don't leak when the line is empty.
	line, _ := rl.completer.Line()
	if line != nil && len(*line) == 0 {
		rl.clearLocalSuggestion()
		rl.ClearAIPrediction()
		if refresh {
			rl.Display.Refresh()
		}
		return true
	}

	rl.aiPredictionMu.Lock()
	suggestion := rl.aiPrediction
	rl.aiPredictionMu.Unlock()

	if suggestion == "" {
		return false
	}

	line, _ = rl.completer.Line()
	if line == nil {
		return false
	}

	currentLine := string(*line)

	// Clear when the suggestion no longer matches the current input (or when the
	// user already typed it all), to avoid stale ghost text reappearing later.
	if currentLine == "" || !strings.HasPrefix(suggestion, currentLine) || len(suggestion) <= len(currentLine) {
		rl.ClearAIPrediction()
		if refresh {
			rl.Display.Refresh()
		}

		return true
	}

	return false
}

//
// Local Suggestion (fast completion-based suggestions without AI) --------------------------
//

// startDelayedLocalSuggestion starts a timer to compute local suggestions after a short delay.
// This provides debouncing to avoid computing suggestions on every keystroke.
func (rl *Shell) startDelayedLocalSuggestion() {
	line, _ := rl.completer.Line()
	if line == nil || len(*line) == 0 {
		rl.clearLocalSuggestion()
		return
	}
	currentLine := string(*line)

	rl.localSuggestionMu.Lock()
	defer rl.localSuggestionMu.Unlock()

	// Cancel any existing timer
	if rl.localSuggestionTimer != nil {
		rl.localSuggestionTimer.Stop()
		rl.localSuggestionTimer = nil
	}

	rl.localSuggestionSeq++
	seq := rl.localSuggestionSeq

	// Use a short delay (50ms) for debouncing
	delay := 50 * time.Millisecond
	if strings.HasSuffix(currentLine, " ") || strings.HasSuffix(currentLine, "\t") {
		delay = 30 * time.Millisecond // Faster after space
	}

	rl.localSuggestionTimer = time.AfterFunc(delay, func() {
		rl.computeLocalSuggestion(seq, currentLine)
	})
}

// computeLocalSuggestion computes and displays a local suggestion.
// Priority: completion candidates > history match
func (rl *Shell) computeLocalSuggestion(seq uint64, currentLine string) {
	rl.localSuggestionMu.Lock()
	if seq != rl.localSuggestionSeq {
		rl.localSuggestionMu.Unlock()
		return
	}
	rl.localSuggestionMu.Unlock()

	// Check if the line has changed since scheduling
	line, _ := rl.completer.Line()
	if line == nil || string(*line) != currentLine {
		return
	}

	var suggestion string

	// Priority 1: Try to get suggestion from completion system
	suggestion = rl.getCompletionSuggestion(currentLine)

	// Priority 2: If no completion, try history match
	if suggestion == "" {
		suggestion = rl.getHistorySuggestion(currentLine)
	}

	if suggestion == "" || suggestion == currentLine {
		rl.clearLocalSuggestion()
		if rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
			rl.Display.RefreshLine()
		}
		return
	}

	// Store and display suggestion
	rl.localSuggestionMu.Lock()
	if seq != rl.localSuggestionSeq {
		rl.localSuggestionMu.Unlock()
		return
	}
	rl.localSuggestion = suggestion
	rl.localSuggestionLine = currentLine
	// Use the existing AI suggestion display mechanism
	rl.SetAISuggestion(suggestion)
	rl.localSuggestionMu.Unlock()

	// Refresh display if the main loop is waiting for input
	if rl.Keys != nil && rl.Keys.IsWaiting() && !rl.Keys.IsReading() {
		rl.Display.RefreshLine()
	}
}

// getCompletionSuggestion gets a suggestion from the completion system.
// Returns the first matching candidate or the common prefix of multiple candidates.
func (rl *Shell) getCompletionSuggestion(currentLine string) string {
	if rl.Completer == nil {
		return ""
	}

	// Get completions for current line
	line := []rune(currentLine)
	cursor := len(line)
	comps := rl.Completer(line, cursor)

	if len(comps.values) == 0 {
		return ""
	}

	// Get the current word prefix
	prefix := ""
	if comps.PREFIX != "" {
		prefix = comps.PREFIX
	} else {
		// Calculate prefix from word boundary
		pos := len(line) - 1
		for pos >= 0 {
			c := line[pos]
			if c == ' ' || c == '\t' {
				break
			}
			pos--
		}
		prefix = string(line[pos+1:])
	}

	// Filter candidates that match the prefix
	var matchingValues []string
	ignoreCase := rl.Config != nil && rl.Config.GetBool("completion-ignore-case")
	for _, v := range comps.values {
		value := v.Value
		matchPrefix := prefix
		if ignoreCase {
			value = strings.ToLower(value)
			matchPrefix = strings.ToLower(matchPrefix)
		}
		if strings.HasPrefix(value, matchPrefix) {
			matchingValues = append(matchingValues, v.Value)
		}
	}

	if len(matchingValues) == 0 {
		return ""
	}

	prefixLen := len([]rune(prefix))
	if prefixLen > len(line) {
		return ""
	}

	// If only one candidate, return it
	if len(matchingValues) == 1 {
		// Build full line with completion
		lineWithoutPrefix := string(line[:len(line)-prefixLen])
		return lineWithoutPrefix + matchingValues[0]
	}

	// Multiple candidates: compute common prefix
	commonPrefix := longestCommonPrefix(matchingValues)
	if len([]rune(commonPrefix)) > prefixLen {
		lineWithoutPrefix := string(line[:len(line)-prefixLen])
		return lineWithoutPrefix + commonPrefix
	}

	return ""
}

// longestCommonPrefix returns the longest common prefix of a slice of strings.
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := []rune(strs[0])
	for _, s := range strs[1:] {
		runes := []rune(s)
		i := 0
		for i < len(prefix) && i < len(runes) && prefix[i] == runes[i] {
			i++
		}
		prefix = prefix[:i]
		if len(prefix) == 0 {
			break
		}
	}
	return string(prefix)
}

// getHistorySuggestion gets a suggestion from command history.
func (rl *Shell) getHistorySuggestion(currentLine string) string {
	suggested := string(rl.History.Suggest(rl.line))
	if suggested == "" || suggested == currentLine {
		return ""
	}
	if !strings.HasPrefix(suggested, currentLine) {
		return ""
	}
	return suggested
}

// hasLocalSuggestion returns true if there's a valid local suggestion for the current line.
func (rl *Shell) hasLocalSuggestion() bool {
	rl.localSuggestionMu.Lock()
	defer rl.localSuggestionMu.Unlock()

	if rl.localSuggestion == "" {
		return false
	}

	line, _ := rl.completer.Line()
	if line == nil {
		return false
	}
	currentLine := string(*line)
	if currentLine == "" {
		return false
	}

	return strings.HasPrefix(rl.localSuggestion, currentLine) &&
		len(rl.localSuggestion) > len(currentLine)
}

// acceptLocalSuggestion accepts the current local suggestion and inserts it.
func (rl *Shell) acceptLocalSuggestion() bool {
	rl.localSuggestionMu.Lock()
	suggestion := rl.localSuggestion
	rl.localSuggestionMu.Unlock()

	if suggestion == "" {
		return false
	}

	line, _ := rl.completer.Line()
	if line == nil {
		return false
	}
	currentLine := string(*line)
	if currentLine == "" {
		rl.clearLocalSuggestion()
		return false
	}

	if !strings.HasPrefix(suggestion, currentLine) || len(suggestion) <= len(currentLine) {
		rl.clearLocalSuggestion()
		return false
	}

	// Only accept when cursor is at end of line
	if rl.cursor.Pos() != rl.line.Len() {
		return false
	}

	// Accept any virtually inserted completion candidate and exit completion mode.
	completion.UpdateInserted(rl.completer)

	// Insert the suggestion suffix
	suffix := suggestion[len(currentLine):]
	for _, r := range suffix {
		rl.line.Insert(rl.cursor.Pos(), r)
		rl.cursor.Inc()
	}

	rl.clearLocalSuggestion()
	rl.Display.Refresh()

	return true
}

// clearLocalSuggestion clears the current local suggestion.
func (rl *Shell) clearLocalSuggestion() {
	rl.localSuggestionMu.Lock()
	defer rl.localSuggestionMu.Unlock()

	if rl.localSuggestionTimer != nil {
		rl.localSuggestionTimer.Stop()
		rl.localSuggestionTimer = nil
	}
	rl.localSuggestionSeq++
	rl.localSuggestion = ""
	rl.localSuggestionLine = ""

	// Clear the displayed suggestion
	rl.Display.ClearAISuggestion()
}
