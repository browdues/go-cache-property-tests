package cache

import "time"

// Operation represents a cache operation
type Operation struct {
	Kind    OpKind
	Key     string
	Value   interface{}
	Expires time.Duration
}

type OpKind int

const (
	OpSet OpKind = iota
	OpGet
	OpDelete
)
