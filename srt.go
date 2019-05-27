package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// grammar:
//
// Subs := {«Sub»}
// Sub := «Number» "\n" «Timespan» "\n" {«Line»} "\n"
// Timespan := «Time» "-->" «Time» /.*$/
// Time := «Digit»«Digit» ":" «Digit»«Digit» ":" «Digit»«Digit» ["," | "."] «Digit»«Digit»«Digit»
// Line := /^[^ ].*$/
// Digit := /[0-9]/
// Number := /[1-9][0-9]*/
//
// We make some allowances to ignore vtt-style stuff.

func ReadSRTFile(fname string) (Subs, error) {
	s, err := ioutil.ReadFile(fname)
	if err != nil {
		return Subs{}, err
	}
	return ParseSRT(string(s))
}

func ParseSRT(s string) (Subs, error) {
	// 
	subs, i, err := parseSubs(s, 0)
	if err == nil && i != len(s) {
		err = fmt.Errorf("trailing data")
	}
	if err != nil {
		n := i + 100
		if n > len(s) {
			n = len(s)
		}
		return Subs{}, fmt.Errorf("error: %s (at character %d around '%s')", err, i+1, s[i:n])
	}
	return subs, nil
}

func parseSubs(s string, i int) (Subs, int, error) {
	subs := Subs{}
	for i < len(s) {
		var sub Sub
		var err error
		i = skipSpace(s, i)
		sub, i, err = parseSub(s, i)
		if err != nil {
			return Subs{}, i, err
		}
		subs.Sub = append(subs.Sub, &sub)
	}
	return subs, i, nil
}

func parseSub(s string, i int) (Sub, int, error) {
	num, i, err := parseNumber(s, i)
	if err != nil {
		return Sub{}, i, err
	}
	i = skipSpace(s, i)
	from, to, i, err := parseTimespan(s, i)
	if err != nil {
		return Sub{}, i, err
	}
	lines, i, err := parseLines(s, i)
	if err != nil {
		return Sub{}, i, err
	}
	return Sub{num, from, to, lines}, i, nil
}

func parseNumber(s string, i int) (int, int, error) {
	i0 := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i0 == i {
		return 0, i0, fmt.Errorf("unexpected character -- expected a digit")
	}
	num, err := strconv.Atoi(s[i0:i])
	if err != nil {
		return 0, i0, fmt.Errorf("programmer error: %s", err)
	}
	return num, i, nil
}

func parseTimespan(s string, i int) (time.Duration, time.Duration, int, error) {
	from, i, err := parseTime(s, i)
	if err != nil {
		return 0, 0, i, err
	}
	i = skipSpace(s, i)
	i, err = skipString(s, i, "-->")
	if err != nil {
		return 0, 0, i, err
	}
	i = skipSpace(s, i)
	to, i, err := parseTime(s, i)
	if err != nil {
		return 0, 0, i, err
	}
	for i < len(s) && s[i-1] != '\n' {
		i++
	}
	return from, to, i, nil
}

func parseTime(s string, i int) (time.Duration, int, error) {
	hour, i, err := parseNumber(s, i)
	if err != nil {
		return 0, i, err
	}
	i, err = skipString(s, i, ":")
	if err != nil {
		return 0, i, err
	}
	min, i, err := parseNumber(s, i)
	if err != nil {
		return 0, i, err
	}
	i, err = skipString(s, i, ":")
	if err != nil {
		return 0, i, err
	}
	sec, i, err := parseNumber(s, i)
	if err != nil {
		return 0, i, err
	}
	i, err = skipStringChoice(s, i, ",", ".")
	if err != nil {
		return 0, i, err
	}
	ms, i, err := parseNumber(s, i)
	if err != nil {
		return 0, i, err
	}
	return time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond, i, nil
}

func parseLines(s string, i int) ([]string, int, error) {
	var lines []string
	for {
		j := i
		for j < len(s) && s[j] != '\n' {
			j++
		}
		if j < len(s) && s[j] == '\n' {
			j++
		}
		l := strings.TrimSpace(s[i:j])
		if l == "" {
			return lines, j, nil
		}
		lines = append(lines, l)
		i = j
	}
}

func skipSpace(s string, i int) int {
	for j, r := range s[i:] {
		if !unicode.IsSpace(r) {
			return i + j
		}
	}
	return len(s)
}

func skipString(s string, i int, t string) (int, error) {
	for j := 0; j < len(t); j++ {
		i_ := j + i
		if i_ >= len(s) {
			return i_, fmt.Errorf("unexpected end of string -- expected '%v'", t[j])
		}
		if s[i_] != t[j] {
			return i_, fmt.Errorf("unexpected character '%v' -- expected '%v'", s[i_], t[j])
		}
	}
	return i + len(t), nil
}

func skipStringChoice(s string, i int, tt ...string) (int, error) {
	for _, t := range tt {
		i_, err := skipString(s, i, t)
		if err == nil {
			return i_, nil
		}
	}
	return i, fmt.Errorf("expected one of: %v", tt)
}
