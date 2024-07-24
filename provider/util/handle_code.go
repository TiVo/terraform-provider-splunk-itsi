package util

type HandleCode int64

const (
	ReturnError HandleCode = 0
	Ignore      HandleCode = 1
	Retry       HandleCode = 2
)
