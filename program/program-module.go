package pords

// ProgramState - Int type for specifiying current state of a program
type ProgramState int

var (
	cancelErr   = ProgramExitError{message: "cancelled"}
	finishedErr = ProgramExitError{message: "finished"}
)

// ProgramModule - interface for cli programs and subprograms
type ProgramModule interface {
	HandleInput(input string) error
	PrintPrompt()
}

// ProgramExitError - Just a custom error for when a program has finished what it needs to do to tell parent program it should be exited.
type ProgramExitError struct {
	message string
}

// pee ( ͡° ͜ʖ ͡°)
func (pee ProgramExitError) Error() string {
	return pee.message
}

// ProgExitErr - The singleton instance of the program exit error
var ProgExitErr = ProgramExitError{}
