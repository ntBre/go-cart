package main

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Molpro struct{}

func (m Molpro) MakeHead() []string {
	return []string{"memory,1125,m",
		"gthresh,energy=1.d-10,zero=1.d-16,oneint=1.d-16,twoint=1.d-16;",
		"nocompress",
		"geomtyp=xyz",
		"angstrom",
		"geometry={"}
}

func (m Molpro) MakeFoot() []string {
	return []string{"}",
		"basis=" + basis,
		"set,charge=" + charge,
		"set,spin=" + spin,
		"hf,accuracy=16,energy=1.0d-10",
		"{" + molproMethod + ",thrden=1.0d-8,thrvar=1.0d-10}"}
}

func (m Molpro) MakeIn(names []string, coords []float64) []string {
	body := make([]string, 0)
	for i, _ := range names {
		tmp := make([]string, 0)
		tmp = append(tmp, names[i])
		for _, c := range coords[3*i : 3*i+3] {
			s := strconv.FormatFloat(c, 'f', 10, 64)
			tmp = append(tmp, s)
		}
		body = append(body, strings.Join(tmp, " "))
	}
	return MakeInput(m.MakeHead(), m.MakeFoot(), body)
}

func (m Molpro) WriteIn(filename string, names []string, coords []float64) {
	lines := m.MakeIn(names, coords)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(filename, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func (m Molpro) ReadOut(filename string) (result float64, err error) {
	runtime.LockOSThread()
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
		return brokenFloat, ErrFileNotFound
	}
	err = ErrEnergyNotFound
	result = brokenFloat
	lines, _ := ReadFile(filename)
	// ASSUME blank file is only created when PBS runs
	// blank file has a single newline
	if len(lines) == 1 {
		if strings.Contains(strings.ToUpper(lines[0]), "ERROR") {
			return result, ErrFileContainsError
		}
		return result, ErrBlankOutput
	}
	for _, line := range lines {
		if strings.Contains(strings.ToUpper(line), "ERROR") {
			return result, ErrFileContainsError
		}
		if strings.Contains(line, energyLine) {
			split := SplitLine(line)
			for i, _ := range split {
				if strings.Contains(split[i], energyLine) {
					// take the thing right after search term
					// not the last entry in the line
					if i+1 < len(split) {
						// assume we found energy so no error
						// from default EnergyNotFound
						err = nil
						result, err = strconv.ParseFloat(split[i+1], 64)
						if err != nil {
							err = ErrEnergyNotParsed
							// false if parse fails
						}
					}
				}
			}
		}
		if strings.Contains(line, molproTerminated) && err != nil {
			err = ErrFinishedButNoEnergy
		}
	}
	runtime.UnlockOSThread()
	return result, err
}
