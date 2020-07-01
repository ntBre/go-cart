package main

import (
	"regexp"
	"strings"
)

// Key is a custom type used as the keys in the Input map
type Key int

// Keys for Input map
const (
	ConcJobKey Key = iota
	DLevelKey
	QueueTypeKey
	ChkIntervalKey
	ProgKey
	GeomKey
	DeltaKey
	MethodKey
	BasisKey
	ChargeKey
	SpinKey
	NumKeys
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
		"MethodKey",
		"BasisKey",
		"ChargeKey",
		"SpinKey",
	}[k]
}

// Regexp consists of an embedded *regexp.Regexp and an associated Key
type Regexp struct {
	*regexp.Regexp
	Name Key
}

// ParseInfile parses filename and loads matching keywords into the
// returned map
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
		Regexp{regexp.MustCompile(`(?i)method=`), MethodKey},
		Regexp{regexp.MustCompile(`(?i)basis=`), BasisKey},
		Regexp{regexp.MustCompile(`(?i)charge=`), ChargeKey},
		Regexp{regexp.MustCompile(`(?i)spin=`), SpinKey},
	}
	geom := regexp.MustCompile(`(?i)geometry={`)
	for i := 0; i < len(lines); {
		if len(lines[i]) < 1 {
			continue
		}
		if lines[i][0] == '#' {
			i++
			continue
		}
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
				if kword.MatchString(lines[i]) {
					split := strings.Split(lines[i], "=")
					keymap[kword.Name] = strings.ToUpper(split[len(split)-1])
				}
			}
			i++
		}
	}
	return keymap
}
