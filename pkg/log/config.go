package log

import "time"

type Config struct {
	Dir    string
	Buffer struct {
		Size    uint64
		Timeout time.Duration
	}
	Errors chan *LogError
}
