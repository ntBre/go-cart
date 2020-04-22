package main

import (
	"errors"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"os"
)

const (
	energyLine  = "energy="
	brokenFloat = 999.999
)

var (
	ErrEnergyNotFound = errors.New("Energy not found in Molpro output")
	ErrFileNotFound = errors.New("Molpro output file not found")
)

type Geometry struct {
	Names  []string
	Coords []float64
}

func ReadFile(filename string) []string {
	lines, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return strings.Split(strings.TrimSpace(string(lines)), "\n")
}

func SplitLine(line string) []string {
	re := regexp.MustCompile(`\s+`)
	trim := strings.TrimSpace(line)
	s := strings.Split(strings.TrimSpace(re.ReplaceAllString(trim, " ")), " ")
	return s
}

func ReadInputXYZ(filename string) Geometry {
	// skip the natoms and comment line in xyz file
	split := ReadFile(filename)
	names := make([]string, 0)
	coords := make([]float64, 0)
	for _, v := range split[2:] {
		s := SplitLine(v)
		if len(s) == 4 {
			names = append(names, s[0])
			for _, c := range s[1:4] {
				f, e := strconv.ParseFloat(c, 64)
				if e != nil {
					panic(e)
				}
				coords = append(coords, f)
			}
		}
	}
	return Geometry{names, coords}
}

func MakeMolproHead() []string {
	return []string{"memory,50,m",
		"nocompress",
		"geomtyp=xyz",
		"angstrom",
		"geometry={"}
}

func MakeMolproFoot() []string {
	return []string{"}",
		"basis=cc-pVTZ-F12",
		"set,charge=0",
		"set,spin",
		"hf",
		"{CCSD(T)-F12}"}
}

func MakeMolproIn(geom *Geometry) []string {
	file := make([]string, 0)
	for _, line := range MakeMolproHead() {
		file = append(file, line)
	}
	for i, _ := range geom.Names {
		tmp := make([]string, 0)
		tmp = append(tmp, geom.Names[i])
		for _, c := range geom.Coords[3*i : 3*i+3] {
			s := strconv.FormatFloat(c, 'f', 10, 64)
			tmp = append(tmp, s)
		}
		file = append(file, strings.Join(tmp, " "))
	}
	for _, line := range MakeMolproFoot() {
		file = append(file, line)
	}
	return file
}

func WriteMolproIn(filename string, geom *Geometry) {
	lines := MakeMolproIn(geom)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(filename, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func ReadMolproOut(filename string) (float64, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return brokenFloat, ErrFileNotFound
	}
	lines := ReadFile(filename)
	for _, line := range lines {
		if strings.Contains(line, energyLine) {
			split := SplitLine(line)
			f, err := strconv.ParseFloat(split[len(split)-1], 64)
			if err != nil {
				panic(err)
			}
			return f, nil
		}
	}
	return brokenFloat, ErrEnergyNotFound
}
