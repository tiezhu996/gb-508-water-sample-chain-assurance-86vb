package service

import "errors"

var (
	ErrInvalidTransition = errors.New("requested status transition is not allowed")
	ErrInvalidInput      = errors.New("business input validation failed")
	ErrUnauthorized      = errors.New("invalid username or password")
	ErrInactiveUser      = errors.New("user account is inactive")
	// ErrBatchNotReceived blocks starting tests on samples whose batch is still
	// planned or collecting.
	ErrBatchNotReceived = errors.New("batch has not been received yet")
	// ErrBatchHasOpenSamples blocks closing a batch while samples under it are
	// not disposed yet.
	ErrBatchHasOpenSamples = errors.New("batch still has undisposed samples")
)
