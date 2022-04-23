package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

// parse netflix-style xml subtitles

func ReadXMLFile(fname string) (Subs, error) {
	s, err := ioutil.ReadFile(fname)
	if err != nil {
		return Subs{}, err
	}
	return ParseXML(string(s))
}

func ParseXML(s string) (Subs, error) {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(s)))
	var subs Subs
	var sub *Sub
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "p" {
				if sub != nil {
					return subs, fmt.Errorf("did not expect <p> inside <p>")
				}
				var id string
				var begin time.Duration = -1
				var end time.Duration = -1
				var err error
				for _, attr := range token.Attr {
					// fmt.Fprintln(os.Stderr, "got attr", attr.Name.Local, attr.Value)
					if attr.Name.Local == "id" {
						id = attr.Value
					}
					if attr.Name.Local == "begin" {
						begin, err = parseXmlTime(attr.Value)
						if err != nil {
							return subs, err
						}
					}
					if attr.Name.Local == "end" {
						end, err = parseXmlTime(attr.Value)
						if err != nil {
							return subs, err
						}
					}
				}
				if id == "" || begin < 0 || end < 0 {
					return subs, fmt.Errorf("<p> missing expected attribute")
				}
				sub = &Sub{
					Number: len(subs.Sub),
					From:   begin,
					To:     end,
				}
			}
		case xml.EndElement:
			if token.Name.Local == "p" {
				subs.Sub = append(subs.Sub, sub)
				sub = nil
			}
		case xml.CharData:
			if sub != nil {
				sub.Lines = append(sub.Lines, strings.TrimSpace(string(token)))
			}
		}
	}
	return subs, nil
}

func parseXmlTime(s string) (time.Duration, error) {
	if len(s) == 0 || s[len(s)-1] != 't' {
		return 0, fmt.Errorf("expected xmlTime to end in 't': %s", s)
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, err
	}
	// assuming xml ticks are 1/10000000s
	// (can verify with `ttp:tickRate="10000000"` header)
	return time.Duration(time.Duration(n) * (time.Second / 10000000)), nil
}
