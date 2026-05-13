package log

import "time"

type Config struct {
	Dir   string
	Write struct {
		Size    int
		Timeout time.Duration
	}
	Read struct {
		Size int
	}
	Errors chan *LogError
}
