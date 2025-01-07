package errs

// Malefic Error
const (
	MaleficErrorPanic uint32 = 1 + iota
	MaleficErrorUnpackError
	MaleficErrorMissbody
	MaleficErrorModuleError
	MaleficErrorModuleNotFound
	MaleficErrorTaskError
	MaleficErrorTaskNotFound
	MaleficErrorTaskOperatorNotFound
	MaleficErrorExtensionNotFound
	MaleficErrorUnexceptBody
)

// task error
const (
	TaskErrorOperatorError       = 2
	TaskErrorNotExpectBody       = 3
	TaskErrorFieldRequired       = 4
	TaskErrorFieldLengthMismatch = 5
	TaskErrorFieldInvalid        = 6
	TaskError                    = 99
)
