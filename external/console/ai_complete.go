package console

// Suggestion represents a single AI-generated command suggestion.
// Kept for backwards compatibility with client code.
type Suggestion struct {
	Command     string // The suggested command
	Description string // Description of what the command does
}
