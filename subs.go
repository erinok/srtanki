package main

import "time"

// Subs is a collection of subtitles.
type Subs struct {
	Sub []Sub
}

// Sub is a single subtitle.
type Sub struct {
	Number   int
	From, To time.Duration
	Lines     []string
}
