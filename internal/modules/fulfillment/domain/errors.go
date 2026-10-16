package domain

import "errors"

var (
	ErrInvalidData = errors.New("fulfillment invalid data")
	ErrNotAllowed  = errors.New("fulfillment operation not allowed")
)
