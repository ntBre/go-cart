package main

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Mopac implements the Program interface
type Mopac struct{}

// MakeHead returns the header for a Mopac input file
func (m Mopac) MakeHead() (headLines []string) {
	return []string{"threads=1 XYZ ANGSTROMS scfcrt=1.D-21 aux(precision=9) " +
		"external=params.dat 1SCF charge=" + charge + " " + mopacMethod,
		"MOLECULE # 1", ""}
}

// MakeFoot returns the empty footer for a Mopac input file
// TODO the interface should just have a MakeIn and these helpers
// should be unexported
func (m Mopac) MakeFoot() (footLines []string) {
	return
}

// MakeIn returns the contents of a Mopac input file
func (m Mopac) MakeIn(names []string, coords []float64) (inputLines []string) {
	body := make([]string, 0)
	for i := range names {
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

// WriteIn uses MakeIn to write a Mopac input file to filename
func (m Mopac) WriteIn(filename string, names []string, coords []float64) {
	lines := m.MakeIn(names, coords)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(filename, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

// ReadOut reads a Mopac output file and returns the resulting energy
func (m Mopac) ReadOut(filename string) (result float64, err error) {
	runtime.LockOSThread()
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
		return brokenFloat, ErrFileNotFound
	}
	err = ErrEnergyNotFound
	result = brokenFloat
	// lines, _ := ReadFile(filename)
	// TODO test if file lost by pbs
	// TODO test for error
	// TODO test for parse failing
	// TODO test for finished but no energy
	return
}
