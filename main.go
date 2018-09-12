// Program srtanki makes flashcards (suitable for import by Anki) given .srt subtitle files a movie file.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	astisub "github.com/asticode/go-astisub"
	"github.com/erinok/jyutping"
)

var (
	srtFile  = flag.String("srt", "", "`SRT` subtitles of movie's spoken audio")
	movFile  = flag.String("mov", "", "extract mp3 clips from `MOVIE`")
	xsrtFile = flag.String("xsrt", "", "`SRT` translated subtitles")
	jp       = flag.Bool("jp", false, "write jyutping romanization of srt")
	xbefore  = flag.Duration("xbefore", 500*time.Millisecond, "include `DUR` time before each audio clip")
	xafter   = flag.Duration("xafter", 2000*time.Millisecond, "include `DUR` time after each audio clip")
	imgWidth = flag.Float64("imgwidth", 1400, "scale imgs to this width")
	numCores = flag.Int("numCore", 2*runtime.NumCPU(), "use up to `CORES` threads while converting audio")
)

var mediaDir, movName string

func clipName(idx int, item *astisub.Item) string {
	return fmt.Sprint(movName, ".", idx+1, ".mp3")
}

func imageName(idx int, item *astisub.Item) string {
	return fmt.Sprint(movName, ".", idx+1, ".jpg")
}

func extractClip(idx int, item *astisub.Item) {
	fname := mediaDir + clipName(idx, item)
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
		fname,
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
}

func extractImage(idx int, item *astisub.Item) {
	fname := mediaDir + imageName(idx, item)
	if stat, err := os.Stat(fname); err == nil && stat.Size() > 0 {
		// image already exists; do nothing
		return
	}
	ss := (item.StartAt + item.EndAt).Seconds() / 2
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.03f", ss),
		"-y", // overwrite existing files
		"-i", *movFile,
		"-vframes", "1",
		"-q:v", "2",
		"-vf", fmt.Sprint("scale=", *imgWidth, ":-1"),
		fname,
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
}

var spacesRegexp = regexp.MustCompile("  +")
var spanRegexp = regexp.MustCompile("<span [^>]*>")

func fmtSub(sub *astisub.Item) string {
	s := join(len(sub.Lines), " ", func(i int) string { return sub.Lines[i].String() })
	s = strings.Replace(s, "\n", "<br/>", -1)
	s = strings.Replace(s, "\t", " ", -1)
	s = strings.Replace(s, "<i>", "", -1)
	s = strings.Replace(s, "</i>", "", -1)
	s = strings.Replace(s, `{\an8}`, "", -1)
	s = spacesRegexp.ReplaceAllString(s, " ")
	s = spanRegexp.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}

func fmtSubs(subs []*astisub.Item) string {
	return join(len(subs), "<br/>", func(i int) string { return fmtSub(subs[i]) })
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

func ankiSound(soundfile string) string { return fmt.Sprintf("[sound:%s]", soundfile) }
func ankiImage(imagefile string) string { return fmt.Sprintf(`<img src="%s">`, imagefile) }

// outputs:
//
// orig text
// [jyutping]
// trans text
// audio
// image
func writeFlashcards(f io.Writer, subs, xsubs *astisub.Subtitles) {
	for i, item := range subs.Items {
		xitems := overlappingSubs(item, xsubs.Items)
		if *jp {
			fmt.Fprint(f,
				fmtSub(item), "\t",
				strings.Replace(jyutping.Convert(fmtSub(item)), "  ", " ", -1), "\t",
				fmtSubs(xitems), "\t",
				ankiSound(clipName(i, item)), "\t",
				ankiImage(imageName(i, item)), "\t",
				"\n")
		} else {
			fmt.Fprint(f,
				fmtSub(item), "\t",
				fmtSubs(xitems), "\t",
				ankiSound(clipName(i, item)), "\t",
				ankiImage(imageName(i, item)), "\t",
				"\n")
		}
	}
}

func main() {
	flag.Parse()
	if *movFile == "" || *srtFile == "" || *xsrtFile == "" {
		fatal("must pass -mov, -srt, and -xsrt")
	}
	mediaDir, movName = filepath.Split(*movFile)
	mediaDir += "media/"
	if err := os.MkdirAll(mediaDir, 0777); err != nil {
		fatal("could not create media directory:", err)
	}
	subs, err := astisub.OpenFile(*srtFile)
	if err != nil {
		fatal(err)
	}
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
	parallelDo(len(subs.Items), *numCores, func(i int) {
		extractClip(i, subs.Items[i])
		extractImage(i, subs.Items[i])
	})
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

func join(n int, sep string, f func(int) string) string {
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = f(i)
	}
	return strings.Join(ss, sep)
}
