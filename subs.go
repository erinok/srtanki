package main

import (
	"fmt"
	"strings"
	"time"
)

// Subs is a collection of subtitles.
type Subs struct {
	Sub []*Sub
}

// Sub is a single subtitle.
type Sub struct {
	Number   int
	From, To time.Duration
	Lines    []string
}

func (s Sub) String() string {
	return strings.Join(s.Lines, "\n")
}

func ReadSubsFile(fname string) (Subs, error) {
	if strings.HasSuffix(fname, ".srt") {
		return ReadSRTFile(fname)
	}
	if strings.HasSuffix(fname, ".xml") {
		return ReadXMLFile(fname)
	}
	return Subs{}, fmt.Errorf("unknown subs file type '%s' -- expected .xml or .srt", fname)
}
