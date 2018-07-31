package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	astisub "github.com/asticode/go-astisub"
)

var (
	srtFile  = flag.String("srt", "", "`SRT` subtitles of movie's spoken audio")
	movFile  = flag.String("mov", "", "extract mp3 clips from `MOVIE`")
	xbefore  = flag.Duration("xbefore", 500*time.Millisecond, "include `DUR` time before each audio clip")
	xafter   = flag.Duration("xafter", 2000*time.Millisecond, "include `DUR` time after each audio clip")
	numCores = flag.Int("numCore", 2*runtime.NumCPU(), "use up to `CORES` threads while converting audio")
)

func extractAudioClip(idx int, item *astisub.Item) {
	clipFile := fmt.Sprint(*movFile, ".", idx, ".mp3")
	ss := (item.StartAt - *xbefore).Seconds()
	t := (item.EndAt + *xafter).Seconds() - ss
	cmd := exec.Command("ffmpeg",
		"-y", // overwrite existing files
		"-i", *movFile,
		"-ss", fmt.Sprintf("%.03f", ss),
		"-t", fmt.Sprintf("%.03f", t),
		clipFile,
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
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
		parallelDo(len(srt.Items), *numCores, func(i int) { 
			extractAudioClip(i+1, srt.Items[i]) 
		})
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

// call f(0), f(1), ..., f(n-1) on separate goroutines; run up to numCores goroutines at once.
func parallelDo(n int, numCores int, f func(int)) {
	wg := sync.WaitGroup{}
	wg.Add(n)
	sema := make(chan struct{}, numCores)
	for i := 0; i < n; i++ {
		i := i // sigh
		go func() {
			sema <- struct{}{}
			f(i)
			<-sema
			wg.Done()
		}()
	}
	wg.Wait()
}
