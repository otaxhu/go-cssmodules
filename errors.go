package cssmodules

import "errors"

// Errors HTML

var (
	ErrClassNotFound = errors.New("css modules class not found")
)

// Errors common

var (
	ErrAlreadyWritten = errors.New("the buffer of this struct has already been written")
)
