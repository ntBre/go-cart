package main

import (
	"errors"
	"hash/maphash"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	energyLine  = "energy="
	brokenFloat = 999.999
)

var (
	ErrEnergyNotFound = errors.New("Energy not found in Molpro output")
	ErrFileNotFound   = errors.New("Molpro output file not found")
	delta             = 0.5
)

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

func ReadInputXYZ(filename string) ([]string, []float64) {
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
	return names, coords
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

func MakeInput(head, foot func() []string, body []string) []string {
	file := make([]string, 0)
	file = append(file, head()...)
	file = append(file, body...)
	file = append(file, foot()...)
	return file
}

func MakeMolproIn(names []string, coords []float64) []string {
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
	return MakeInput(MakeMolproHead, MakeMolproFoot, body)
}

func WriteMolproIn(filename string, names []string, coords []float64) {
	lines := MakeMolproIn(names, coords)
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

func Basename(filename string) string {
	file := path.Base(filename)
	re := regexp.MustCompile(path.Ext(file))
	basename := re.ReplaceAllString(file, "")
	return basename
}

func Qsubmit(filename string) int {
	pbsname := filename + ".pbs"
	cmd := exec.Command("qsub", pbsname)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	b := Basename(string(out))
	i, _ := strconv.Atoi(b)
	return i
}

func MakePBSHead() []string {
	return []string{"#!/bin/sh",
		"#PBS -N go-cart",
		"#PBS -S /bin/bash",
		"#PBS -j oe",
		"#PBS -W umask=022",
		"#PBS -l walltime=00:30:00",
		"#PBS -l ncpus=1",
		"#PBS -l mem=50mb",
		"module load intel",
		"module load mvapich2",
		"module load pbspro",
		"export PATH=/usr/local/apps/molpro/2015.1.35/bin:$PATH",
		"export WORKDIR=$PBS_O_WORKDIR",
		"export TMPDIR=/tmp/$USER/$PBS_JOBID",
		"cd $WORKDIR",
		"mkdir -p $TMPDIR",
		"date"}
}

func MakePBSFoot() []string {
	return []string{"date",
		"rm -rf $TMPDIR"}
}

func MakePBS(filename string) []string {
	body := []string{"molpro -t 1 " + filename}
	return MakeInput(MakePBSHead, MakePBSFoot, body)
}

func WritePBS(pbsfile, molprofile string) {
	lines := MakePBS(molprofile)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func Make2D(i, j int) []Job {
	if i == j {
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []Job{Job{1, HashName(), []int{i, i}, "queued", 0},
			Job{2, "E0", []int{0}, "done", 0},
			Job{1, HashName(), []int{-i, -i}, "queued", 0}}
	} else {
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []Job{Job{1, HashName(), []int{i, j}, "queued", 0},
			Job{1, HashName(), []int{i, -j}, "queued", 0},
			Job{1, HashName(), []int{-i, j}, "queued", 0},
			Job{1, HashName(), []int{-i, -j}, "queued", 0}}
	}

}

func Derivative(dims ...int) []Job {
	switch len(dims) {
	case 2:
		return Make2D(dims[0], dims[1])
	}
	return []Job{Job{}}
}

type Job struct {
	Coeff   int
	Name    string
	Steps   []int
	Status  string
	Retries int
}

func Step(coords []float64, steps ...int) []float64 {
	var c = make([]float64, len(coords))
	copy(c, coords)
	for _, v := range steps {
		if v < 0 {
			v = -1 * v
			c[v-1] = c[v-1] - delta
		} else {
			c[v-1] += delta
		}
	}
	return c
}

func HashName() string {
	var h maphash.Hash
	h.SetSeed(maphash.MakeSeed())
	return "job" + strconv.FormatUint(h.Sum64(), 16)
}

// func main() {
// 	// loop through every position in the 9x9 matrix of
// 	// force constants
// 	// let the derivative function handle how to call
// 	// each version of the derivative with just the indices
// 	// derivative should build a []Job for each derivative
// 	// with the associated geometries
// 	jobs := make([]Job, 0)
// 	fcs := make([][]float64, 0) // append here on outer read loop
// 	fcRow := make([]float64, 0) // read into here on inner loop
// 	for i := 0; i < len(coords); i++ {
// 		for j := 0; j <= i; j++ {
// 			// maybe these are go routines -> channel?
// 			jobs = append(jobs, Derivative(i, j))
// 		}
// 	}
// 	// then have another loop for running the jobs/reading output
// 	// ch <- ? reading from the channel
// }
