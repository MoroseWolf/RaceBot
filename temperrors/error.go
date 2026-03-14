package temperrors

import (
	"errors"
)

var (
	ErrEmptyList = errors.New("empty list")
	ErrParse     = errors.New("parse error")
)
