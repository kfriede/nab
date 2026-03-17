package output

// AppError represents a structured error for both human and agent consumption.
type AppError struct {
	Code     string // Machine-readable error code (e.g., "AUTH_EXPIRED", "NOT_FOUND")
	Message  string // Human-readable error message
	Guidance string // Actionable next-step instructions (for humans and agents)
}

// Common error codes
const (
	ErrCodeGeneral    = "GENERAL_ERROR"
	ErrCodeAuth       = "AUTH_ERROR"
	ErrCodeNotFound   = "NOT_FOUND"
	ErrCodeConflict   = "CONFLICT"
	ErrCodeValidation = "VALIDATION_ERROR"
	ErrCodeNetwork    = "NETWORK_ERROR"
	ErrCodeConfig     = "CONFIG_ERROR"
	ErrCodeRateLimit  = "RATE_LIMIT"
)

// Exit codes matching our spec
const (
	ExitSuccess  = 0
	ExitGeneral  = 1
	ExitAuth     = 2
	ExitNotFound = 3
	ExitConflict = 4
)

// Error implements the error interface so AppError can be used as an error.
func (e AppError) Error() string {
	return e.Message
}

// ExitCode returns the appropriate exit code for this error.
func (e AppError) ExitCode() int {
	switch e.Code {
	case ErrCodeAuth:
		return ExitAuth
	case ErrCodeNotFound:
		return ExitNotFound
	case ErrCodeConflict, ErrCodeValidation:
		return ExitConflict
	default:
		return ExitGeneral
	}
}

// NewError creates a new AppError.
func NewError(code, message, guidance string) AppError {
	return AppError{
		Code:     code,
		Message:  message,
		Guidance: guidance,
	}
}

// NewAuthError creates an authentication error.
func NewAuthError(message string) AppError {
	return AppError{
		Code:     ErrCodeAuth,
		Message:  message,
		Guidance: "Run `nab login` to authenticate, or set the NAB_TOKEN environment variable.",
	}
}

// NewNotFoundError creates a not-found error.
func NewNotFoundError(resource, id string) AppError {
	return AppError{
		Code:     ErrCodeNotFound,
		Message:  resource + " not found: " + id,
		Guidance: "Run `nab " + resource + " list` to see available " + resource + "s.",
	}
}

// NewValidationError creates a validation error.
func NewValidationError(message string) AppError {
	return AppError{
		Code:     ErrCodeValidation,
		Message:  message,
		Guidance: "Check the command flags and try again. Run with --help for usage details.",
	}
}
