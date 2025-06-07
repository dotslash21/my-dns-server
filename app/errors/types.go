package errors

import (
	"errors"
	"fmt"
	"runtime"
)

type ErrorType int

const (
	// ErrorTypeServer Represents Server errors
	ErrorTypeServer ErrorType = iota
)

// Error represents a custom error with additional context
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
	Stack   string
}

// New creates a new Error with stack trace
func New(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Stack:   getStack(),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Cause:   err,
		Stack:   getStack(),
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap implements the unwrap interface
func (e *Error) Unwrap() error {
	return e.Cause
}

// IsServerError checks if an error is a server error
func IsServerError(err error) bool {
	var e *Error
	if ok := As(err, &e); ok {
		return e.Type == ErrorTypeServer
	}
	return false
}

// getStack returns the stack trace as a string
func getStack() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var stack string
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		stack += fmt.Sprintf("\n%s:%d - %s", frame.File, frame.Line, frame.Function)
	}
	return stack
}

// As implements the As interface for error handling
func As(err error, target interface{}) bool {
	return errors.As(err, &target)
}

// Is implements the Is interface for error handling
func Is(err, target error) bool {
	return errors.Is(err, target)
}
