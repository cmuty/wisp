package mediakeys

type Action int

const (
	TogglePause Action = iota + 1
	Next
	Previous
	Stop
)
