// Program srtanki makes flashcards (suitable for import by Anki) given .srt subtitle files and a movie file.
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
	"time"

	"github.com/erinok/jyutping"
)

var (
	srtFile     = flag.String("srt", "", "`SRT` subtitles of movie's spoken audio")
	movFile     = flag.String("mov", "", "extract mp3 clips from `MOVIE`")
	xsrtFile    = flag.String("xsrt", "", "`SRT` translated subtitles")
	jp          = flag.Bool("jp", false, "write jyutping romanization of srt")
	ruby        = flag.Bool("ruby", false, "write jyutping ruby annotations")
	colorize    = flag.Bool("colorize", false, "write chinese characters wrapped in tone spans (which can be drawn in colors in anki)")
	xbefore     = flag.Duration("xbefore", 500*time.Millisecond, "include `DUR` time before each audio clip")
	xafter      = flag.Duration("xafter", 2000*time.Millisecond, "include `DUR` time after each audio clip")
	noMerge     = flag.Bool("noMerge", false, "don't merge subtitles")
	maxMergeGap = flag.Duration("maxMergeGap", 3*time.Second, "allow subtitles that are part of the same sentence to be merged if gap between them is less than `DURATION`")
	imgWidth    = flag.Float64("imgwidth", 1400, "scale imgs to this width")
	numCores    = flag.Int("numCore", 2*runtime.NumCPU(), "use up to `CORES` threads while converting audio")

	canto = flag.Bool("canto", false, "alias for -jp -ruby -colorize -noMerge")
)

var mediaDir, movName string

func clipName(idx int, item *Sub) string {
	return fmt.Sprint(movName, ".", idx+1, ".mp3")
}

func imageName(idx int, item *Sub) string {
	return fmt.Sprint(movName, ".", idx+1, ".jpg")
}

func extractClip(idx int, item *Sub) {
	nm := clipName(idx, item)
	fname := mediaDir + nm
	if stat, err := os.Stat(fname); err == nil && stat.Size() > 0 {
		// clip already exists; do nothing
		return
	}
	tmp := mediaDir + "tmp." + nm
	ss := (item.From - *xbefore).Seconds()
	t := (item.To + *xafter).Seconds() - ss
	cmd := exec.Command("nice", "ffmpeg",
		"-y", // overwrite existing files
		"-i", *movFile,
		"-ss", fmt.Sprintf("%.03f", ss),
		"-t", fmt.Sprintf("%.03f", t),
		tmp,
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
	if err = os.Rename(tmp, fname); err != nil {
		fatal("error moving temporary file into final location:", err)
	}
}

func extractImage(idx int, item *Sub) {
	nm := imageName(idx, item)
	fname := mediaDir + nm
	if stat, err := os.Stat(fname); err == nil && stat.Size() > 0 {
		// image already exists; do nothing
		return
	}
	tmp := mediaDir + "tmp." + nm
	ss := (item.From + item.To).Seconds() / 2
	cmd := exec.Command("nice", "ffmpeg",
		"-ss", fmt.Sprintf("%.03f", ss),
		"-y", // overwrite existing files
		"-i", *movFile,
		"-vframes", "1",
		"-q:v", "2",
		"-vf", fmt.Sprint("scale=", *imgWidth, ":-1"),
		tmp,
	)
	fmt.Println(">", strings.Join(cmd.Args, " "))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fatal("error running ffmpeg:\n", string(buf))
	}
	if err = os.Rename(tmp, fname); err != nil {
		fatal("error moving temporary file into final location:", err)
	}
}

var spacesRegexp = regexp.MustCompile(`  +`)
var spanRegexp = regexp.MustCompile(`<span [^>]*>`)
var newlineRegexp = regexp.MustCompile(`
([^-])`)

func fmtSub(sub *Sub) string {
	s := strings.Join(sub.Lines, "\n")
	if !*noMerge {
		s = newlineRegexp.ReplaceAllString(s, " $1")
	}
	s = strings.Replace(s, "\n", "<br/>", -1)
	s = strings.Replace(s, "\t", " ", -1)
	s = strings.Replace(s, "<i>", "", -1)
	s = strings.Replace(s, "</i>", "", -1)
	s = strings.Replace(s, `{\an8}`, "", -1)
	s = strings.Replace(s, `"`, "&quot", -1)
	s = spacesRegexp.ReplaceAllString(s, " ")
	s = spanRegexp.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}

func gentleFmtSub(sub *Sub) string {
	s := strings.Join(sub.Lines, "\n")
	s = strings.TrimSpace(s)
	return s
}

func fmtSubs(subs []*Sub) string {
	return join(len(subs), "<br/>", func(i int) string { return fmtSub(subs[i]) })
}

// return the items in subs that overlap sub
func overlappingSubs(sub *Sub, subs []*Sub) []*Sub {
	i := 0
	for i < len(subs) && !(sub.From <= subs[i].To) {
		i++
	}
	j := i
	for j < len(subs) && !(sub.To < subs[j].From) {
		j++
	}
	return subs[i:j]
}

var unendedSentenceRegexp = regexp.MustCompile(`[\pL,]$`)
var allCapsRegexp = regexp.MustCompile(`^\pLu*$`)

// should the two subtitles be merged?
func shouldMerge(s, t *Sub) bool {
	if *noMerge {
		return false
	}
	a := fmtSub(s)
	a = a[strings.LastIndexByte(a, '\n')+1:]
	if allCapsRegexp.MatchString(a) {
		return false
	}
	if t.From-s.To > *maxMergeGap {
		return false
	}
	return unendedSentenceRegexp.MatchString(a)
}

// return a new subtitle merging the two subtitles
func merge(s, t *Sub) *Sub {
	return &Sub{s.Number, s.From, t.To, append(s.Lines, t.Lines...)}
}

// merge all subtitles that should be merged
func mergeSubs(subs Subs) Subs {
	var ss Subs
	i := 0
	for i < len(subs.Sub) {
		s := subs.Sub[i]
		j := i + 1
		for j < len(subs.Sub) && shouldMerge(s, subs.Sub[j]) {
			s = merge(s, subs.Sub[j])
			j++
		}
		ss.Sub = append(ss.Sub, s)
		i = j
	}
	return ss
}

func ankiSound(soundfile string) string { return fmt.Sprintf("[sound:%s]", soundfile) }
func ankiImage(imagefile string) string { return fmt.Sprintf(`<img src="%s">`, imagefile) }

func cleanJyutping(jp string) string {
	jp = strings.Replace(jp, "  ", " ", -1)
	jp = strings.Replace(jp, `"`, "&quot", -1)
	return jp
}

// outputs:
//
// image
// audio
// orig text
// trans text
// [jyutping]
// [ruby]
// [origcolor]
func writeFlashcards(f io.Writer, subs, xsubs Subs) {
	for i, item := range subs.Sub {
		xitems := overlappingSubs(item, xsubs.Sub)
		cols := []string{
			ankiImage(imageName(i, item)),
			ankiSound(clipName(i, item)),
			fmtSub(item),
			fmtSubs(xitems),
		}
		if *jp {
			cols = append(cols, cleanJyutping(jyutping.Convert(fmtSub(item))))
		}
		if *ruby {
			cols = append(cols, jyutping.ConvertRuby(gentleFmtSub(item)))
		}
		if *colorize {
			cols = append(cols, jyutping.ColorizeChars(fmtSub(item)))
			// cols = append(cols, "test")
		}
		fmt.Fprintln(f, strings.Join(cols, "\t"))
	}
}

func ezMovFile() string {
	ff, err := filepath.Glob("*.mov")
	if err != nil {
		fatal(err)
	}
	if len(ff) == 0 {
		fatal("no matching .mov files")
	}
	if len(ff) > 1 {
		fatal("ambiguous movie -- >1 .mov files")
	}
	return ff[0]
}

func main() {
	flag.Parse()
	if *movFile == "" || *srtFile == "" || *xsrtFile == "" {
		fatal("must pass -mov, -srt, and -xsrt")
	}
	if *canto {
		*jp = true
		*ruby = true
		*colorize = true
		*noMerge = true
	}
	
	mediaDir, movName = filepath.Split(*movFile)
	mediaDir += "media/"
	if err := os.MkdirAll(mediaDir, 0777); err != nil {
		fatal("could not create media directory:", err)
	}
	subs, err := ReadSRTFile(*srtFile)
	if err != nil {
		fatal(err)
	}
	xsubs, err := ReadSRTFile(*xsrtFile)
	if err != nil {
		fatal(err)
	}
	subs = mergeSubs(subs)
	f, err := os.Create(*movFile + ".cards.tsv")
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	writeFlashcards(f, subs, xsubs)
	parallelDo(len(subs.Sub), *numCores, func(i int) {
		extractClip(i, subs.Sub[i])
		extractImage(i, subs.Sub[i])
	})
	if false {
		for _, item := range subs.Sub {
			fmt.Println(item.String())
		}
	}
}

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
