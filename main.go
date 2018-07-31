package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	astisub "github.com/asticode/go-astisub"
)

var (
	srtFile = flag.String("srt", "", "`SRT` subtitles of movie's spoken audio")
	movFile = flag.String("mov", "", "extract mp3 clips from `MOVIE`")
	xbefore = flag.Duration("xbefore", 500*time.Millisecond, "include `DUR` time before each audio clip")
	xafter  = flag.Duration("xafter", 2000*time.Millisecond, "include `DUR` time after each audio clip")
)

func extractAudioClip(idx int, item *astisub.Item) {
	clipFile := fmt.Sprint(*movFile, ".", idx, ".mp3")
	ss := (item.StartAt - *xbefore).Seconds()
	t := (item.EndAt + *xafter).Seconds() - ss
	cmd := []string{
		"ffmpeg",
		"-i", *movFile,
		"-ss", fmt.Sprintf("%.03f", ss),
		"-t", fmt.Sprintf("%.03f", t),
		clipFile,
	}
	fmt.Println(strings.Join(cmd, " "))
}

// outputs:
//
// orig text
// trans text
// audio

func main() {
	flag.Parse()
	srt, err := astisub.OpenFile(*srtFile)
	if err != nil {
		fatal(err)
	}
	if *movFile != "" {
		for idx, item := range srt.Items {
			extractAudioClip(idx+1, item)
		}
	}
	if false {
		for _, item := range srt.Items {
			fmt.Println(item.String())
		}
	}
}

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
