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
	MethodKey
	BasisKey
	ChargeKey
	SpinKey
//	NumKeys -- Unused variable.
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
// Unnecessary type declarations.
	Keywords := []Regexp{
		{regexp.MustCompile(`(?i)concjobs=`), ConcJobKey},
		{regexp.MustCompile(`(?i)derivative=`), DLevelKey},
		{regexp.MustCompile(`(?i)queuetype=`), QueueTypeKey},
		{regexp.MustCompile(`(?i)chkinterval=`), ChkIntervalKey},
		{regexp.MustCompile(`(?i)program=`), ProgKey},
		{regexp.MustCompile(`(?i)delta=`), DeltaKey},
		{regexp.MustCompile(`(?i)method=`), MethodKey},
		{regexp.MustCompile(`(?i)basis=`), BasisKey},
		{regexp.MustCompile(`(?i)charge=`), ChargeKey},
		{regexp.MustCompile(`(?i)spin=`), SpinKey},
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
