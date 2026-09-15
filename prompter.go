package task

import "github.com/go-task/task/v3/errors"

// ErrPromptCancelled is returned by a Prompter when the user dismissed the
// question rather than answering it. Task stops the run, as it does when a
// prompt is declined on the terminal.
var ErrPromptCancelled = errors.New("task: prompt cancelled")

// Prompter answers the questions Task needs to ask while it runs: the
// confirmation a task declares with "prompt", and the value of a required
// variable that was not supplied. Assign one to Executor.Prompter.
//
// A client that draws its own display should set one, or Task has nowhere to
// ask and refuses to run anything that needs an answer. A client that leaves
// Task's own streams alone need not: Task asks on the terminal as it always
// has.
//
// Methods are called from the goroutines running the tasks and must be safe for
// concurrent use. A call blocks the task that asked until it returns.
type Prompter interface {
	// Confirm asks whether to run a task that declares "prompt". Returning
	// false stops the task, as declining on the terminal does.
	Confirm(task, message string) (bool, error)

	// Ask asks for a required variable that was not supplied. The answer
	// becomes the variable's value.
	//
	// Task only asks for strings today, so a client should return one. The
	// return is any because a variable's value is any: when Task's schema grows
	// to declare types, VarRequest will say which is wanted and this will carry
	// it.
	Ask(VarRequest) (any, error)
}

// VarRequest is a required variable Task needs a value for.
type VarRequest struct {
	// Task is the name of the task that needs it, as written in the Taskfile,
	// which is the key GetTask takes.
	Task string
	Name string
	Type VarType
}

// VarType is what a variable's value must be. Task always sets one.
//
// The set grows as Task's "requires" schema does. A client should handle an
// unrecognised type by returning an error rather than guessing: a wrong guess
// produces a value the task will act on. Adding a type breaks no client, since
// only Task can add one.
type VarType interface {
	isVarType()
}

// StringVar is free text.
type StringVar struct{}

// EnumVar is one of a fixed set of values.
//
// A Taskfile can declare an enum by reference, and a reference that does not
// resolve before the run falls back to free text, so Task may ask for a
// StringVar where the Taskfile said "enum".
type EnumVar struct {
	Options []string
}

func (StringVar) isVarType() {}
func (EnumVar) isVarType()   {}
