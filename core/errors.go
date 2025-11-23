package core

import "errors"

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrPRExists    = errors.New("PR already exists")
	ErrPRMerged    = errors.New("cannot modify merged PR")
	ErrNotAssigned = errors.New("reviewer is not assigned")
	ErrNoCandidate = errors.New("no active candidate available")
	ErrNotFound    = errors.New("resource not found")
)
