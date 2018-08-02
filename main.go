package main

import (
	"flag"
	"fmt"
	"io"
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
	xsrtFile = flag.String("xsrt", "", "`SRT` translated subtitles")
	xbefore  = flag.Duration("xbefore", 500*time.Millisecond, "include `DUR` time before each audio clip")
	xafter   = flag.Duration("xafter", 2000*time.Millisecond, "include `DUR` time after each audio clip")
	numCores = flag.Int("numCore", 2*runtime.NumCPU(), "use up to `CORES` threads while converting audio")
)

func clipFname(idx int, item *astisub.Item) string {
	return fmt.Sprint(*movFile, ".", idx+1, ".mp3")
}

func extractAudioClip(idx int, item *astisub.Item) {
	fname := clipFname(idx, item)
	if stat, err := os.Stat(fname); err == nil && stat.Size() > 0 {
		// clip already exists; do nothing
		return
	}
	ss := (item.StartAt - *xbefore).Seconds()
	t := (item.EndAt + *xafter).Seconds() - ss
	cmd := exec.Command("ffmpeg",
		"-y", // overwrite existing files
		"-i", *movFile,
		"-ss", fmt.Sprintf("%.03f", ss),
		"-t", fmt.Sprintf("%.03f", t),
		clipFname(idx, item),
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
}

func fmtSub(sub *astisub.Item) string {
	s := sub.String()
	s = strings.Replace(s, "\n", "<br/>", -1)
	s = strings.Replace(s, "\t", " ", -1)
	return s
}

func fmtSubs(subs []*astisub.Item) string {
	var s string
	for i, sub := range subs {
		if i > 0 {
			s += "<br/>"
		}
		s += fmtSub(sub)
	}
	return s
}

// return the items in subs that overlap sub
func overlappingSubs(sub *astisub.Item, subs []*astisub.Item) []*astisub.Item {
	i := 0
	for i < len(subs) && !(sub.StartAt <= subs[i].EndAt) {
		i++
	}
	j := i
	for j < len(subs) && !(sub.EndAt < subs[j].StartAt) {
		j++
	}
	return subs[i:j]
}

// outputs:
//
// orig text
// trans text
// audio
func writeFlashcards(f io.Writer, subs, xsubs *astisub.Subtitles) {
	for i, item := range subs.Items {
		xitems := overlappingSubs(item, xsubs.Items)
		fmt.Fprintln(f, fmtSub(item), "\t", fmtSubs(xitems), "\t", clipFname(i, item))
	}
}

func main() {
	flag.Parse()
	if *movFile == "" || *srtFile == "" {
		fatal("must pass -mov, -srt, and -xrt")
	}
	subs, err := astisub.OpenFile(*srtFile)
	if err != nil {
		fatal(err)
	}
	parallelDo(len(subs.Items), *numCores, func(i int) {
		extractAudioClip(i, subs.Items[i])
	})
	xsubs, err := astisub.OpenFile(*xsrtFile)
	if err != nil {
		fatal(err)
	}
	f, err := os.Create(*movFile + ".cards.tsv")
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	writeFlashcards(f, subs, xsubs)
	if false {
		for _, item := range subs.Items {
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
