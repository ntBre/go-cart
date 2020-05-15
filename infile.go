package main

import (
	"regexp"
	"strings"
)

type Key int

const (
	ConcJobKey Key = iota
	DLevelKey
	QueueTypeKey
	ChkIntervalKey
	ProgKey
	GeomKey
	DeltaKey
)

func (k Key) String() string {
	return [...]string{
		"ConcJobKey",
		"DLevelKey",
		"QueueTypeKey",
		"ChkIntervalKey",
		"ProgKey",
		"GeomKey",
		"DeltaKey",
	}[k]
}

type Regexp struct {
	Expr *regexp.Regexp
	Name Key
}

func ParseInfile(filename string) map[Key]string {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	keymap := map[Key]string{}

	Keywords := []Regexp{
		Regexp{regexp.MustCompile(`(?i)concjobs=`), ConcJobKey},
		Regexp{regexp.MustCompile(`(?i)derivative=`), DLevelKey},
		Regexp{regexp.MustCompile(`(?i)queuetype=`), QueueTypeKey},
		Regexp{regexp.MustCompile(`(?i)chkinterval=`), ChkIntervalKey},
		Regexp{regexp.MustCompile(`(?i)program=`), ProgKey},
		Regexp{regexp.MustCompile(`(?i)delta=`), DeltaKey},
	}
	geom := regexp.MustCompile(`(?i)geometry={`)
	for i := 0; i < len(lines); {
		if geom.MatchString(lines[i]) {
			i++
			geomlines := make([]string, 0)
			for !strings.Contains(lines[i], "}") {
				geomlines = append(geomlines, lines[i])
				i++
			}
			keymap[GeomKey] = strings.Join(geomlines, "\n")
		} else {
			for _, kword := range Keywords {
				if kword.Expr.MatchString(lines[i]) {
					split := strings.Split(lines[i], "=")
					keymap[kword.Name] = strings.ToUpper(split[len(split)-1])
				}
			}
			i++
		}
	}
	return keymap
}
