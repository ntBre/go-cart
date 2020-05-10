package main

import (
	"errors"
	"fmt"
	"hash/maphash"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	energyLine = "energy="
	angborh    = 0.529177249
	progName   = "go-cart"
	RTMIN      = 35
	RTMAX      = 64
)

var (
	ErrEnergyNotFound = errors.New("Energy not found in Molpro output")
	ErrFileNotFound   = errors.New("Molpro output file not found")
	delta             = 0.005
	progress          = 1
	brokenFloat       = math.NaN()
	// may need to adjust this if jobs can reasonably take longer than a minute
	timeBeforeRetry = time.Second * 60
)

func ReadFile(filename string) ([]string, error) {
	lines, err := ioutil.ReadFile(filename)
	return strings.Split(strings.TrimSpace(string(lines)), "\n"), err
}

func SplitLine(line string) []string {
	re := regexp.MustCompile(`\s+`)
	trim := strings.TrimSpace(line)
	s := strings.Split(strings.TrimSpace(re.ReplaceAllString(trim, " ")), " ")
	return s
}

func ReadInputXYZ(filename string) ([]string, []float64) {
	// skip the natoms and comment line in xyz file
	split, _ := ReadFile(filename)
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

func MakeInput(head, foot, body []string) []string {
	file := make([]string, 0)
	file = append(file, head...)
	file = append(file, body...)
	file = append(file, foot...)
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
	return MakeInput(MakeMolproHead(), MakeMolproFoot(), body)
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
	runtime.LockOSThread()
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
		return brokenFloat, ErrFileNotFound
	}
	// keeps giving error file not found
	// even though I just checked if the file exists
	// so instead of panic in ReadFile, just return the error
	// and disregard here
	lines, _ := ReadFile(filename)
	for _, line := range lines {
		if strings.Contains(line, energyLine) {
			split := SplitLine(line)
			for i, _ := range split {
				if strings.Contains(split[i], energyLine) {
					// take the thing right after search term
					// not the last entry in the line
					if i+1 >= len(split) {
						runtime.UnlockOSThread()
						return brokenFloat, ErrEnergyNotFound
					}
					f, err := strconv.ParseFloat(split[i+1], 64)
					runtime.UnlockOSThread()
					return f, err
					// return err here to catch problem with conversion
				}
			}
		}
	}
	runtime.UnlockOSThread()
	return brokenFloat, ErrEnergyNotFound
}

func Basename(filename string) string {
	file := path.Base(filename)
	re := regexp.MustCompile(path.Ext(file))
	basename := re.ReplaceAllString(file, "")
	return basename
}

func Qsubmit(filename string) int {
	// -f option to run qsub in foreground
	out, err := exec.Command("qsub", "-f", filename).Output()
	for err != nil {
		time.Sleep(time.Second)
		out, err = exec.Command("qsub", filename).Output()
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
		"#PBS -o /dev/null",
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

type GarbageHeap struct {
	Heap []string // list of basenames
}

func (g *GarbageHeap) Dump() []string {
	dump := make([]string, 0)
	for _, v := range g.Heap {
		dump = append(dump, "rm "+v+"*")
	}
	g.Heap = []string{}
	return dump
}

func MakePBSFoot(count int, dump *GarbageHeap) []string {
	num := strconv.Itoa(count)
	return []string{"ssh -t maple pkill -" + num + " " + progName,
		strings.Join(dump.Dump(), "\n"), "rm -rf $TMPDIR"}
}

func MakePBS(filename string, count int, dump *GarbageHeap) []string {
	body := []string{"molpro -t 1 " + filename}
	return MakeInput(MakePBSHead(), MakePBSFoot(count, dump), body)
}

func WritePBS(pbsfile, molprofile string, count int, dump *GarbageHeap) {
	lines := MakePBS(molprofile, count, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func Make2D(i, j int) []Job {
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i}, []int{i, i}, "queued", 0, 0},
			Job{-2, "E0", 0, 0, []int{}, []int{i, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i}, []int{i, i}, "queued", 0, 0}}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, j}, []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, -j}, []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, j}, []int{i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -j}, []int{i, j}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Make3D(i, j, k int) []Job {
	switch {
	case i == j && i == k:
		// E(+i+i+i) - 3*E(i) + 3*E(-i) -E(-i-i-i) / (2d)^3
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, i}, []int{i, i, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{i}, []int{i, i, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{-i}, []int{i, i, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -i}, []int{i, i, i}, "queued", 0, 0}}
	case i == j && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, k}, []int{i, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{k}, []int{i, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, k}, []int{i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, -k}, []int{i, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-k}, []int{i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -k}, []int{i, i, k}, "queued", 0, 0}}
	case i == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, j}, []int{i, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{j}, []int{i, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, j}, []int{i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, -j}, []int{i, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-j}, []int{i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -j}, []int{i, i, j}, "queued", 0, 0}}
	case j == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, []int{j, j, i}, []int{j, j, i}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i}, []int{j, j, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, i}, []int{j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, -i}, []int{j, j, i}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-i}, []int{j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, -i}, []int{j, j, i}, "queued", 0, 0}}
	case i != j && i != k && j != k:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, -j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, -j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -j, -k}, []int{i, j, k}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Make4D(i, j, k, l int) []Job {
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, i, i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{-4, HashName(), 0, 0, []int{i, i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{6, "E0", 0, 0, []int{}, []int{i, i, i, i}, "queued", 0, 0},
			Job{-4, HashName(), 0, 0, []int{-i, -i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -i, -i}, []int{i, i, i, i}, "queued", 0, 0}}
	// 3 and 1
	case i == j && i == k && i != l:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{-i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{-i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -i, -l}, []int{i, i, i, l}, "queued", 0, 0}}
	case i == j && i == l && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{-i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{-i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -i, -k}, []int{i, i, i, k}, "queued", 0, 0}}
	case i == k && i == l && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{-i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{-i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -i, -j}, []int{i, i, i, j}, "queued", 0, 0}}
	case j == k && j == l && j != i:
		return []Job{
			Job{1, HashName(), 0, 0, []int{j, j, j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{-j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, -j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, []int{j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, []int{-j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, -j, -i}, []int{j, j, j, i}, "queued", 0, 0}}
	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		// x -> i, y -> k, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, -k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -k, -l}, []int{i, i, k, l}, "queued", 0, 0}}
	case i == k && i != j && i != l && j != l:
		// x -> i, y -> j, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, -j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -j, -l}, []int{i, i, j, l}, "queued", 0, 0}}
	case i == l && i != j && i != k && j != k:
		// x -> i, y -> k, z -> j
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, -k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, -k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, i, k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -i, k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -k, -j}, []int{i, i, k, j}, "queued", 0, 0}}
	case j == k && j != i && j != l && i != l:
		// x -> j, y -> i, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, []int{j, j, i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, -i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, -i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{j, j, -i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, -i, -l}, []int{j, j, i, l}, "queued", 0, 0}}
	case j == l && j != i && j != k && i != k:
		// x -> j, y -> i, z -> k
		return []Job{
			Job{1, HashName(), 0, 0, []int{j, j, i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, -i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, -i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{j, j, i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-j, -j, i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{j, j, -i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-j, -j, -i, -k}, []int{j, j, i, k}, "queued", 0, 0}}
	case k == l && k != i && k != j && i != j:
		// x -> k, y -> i, z -> j
		return []Job{
			Job{1, HashName(), 0, 0, []int{k, k, i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-k, -k, i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{k, k, -i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{-i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-k, -k, -i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{k, k, i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, []int{i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-k, -k, i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{k, k, -i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-k, -k, -i, -j}, []int{k, k, i, j}, "queued", 0, 0}}
	// 2 and 2
	case i == j && k == l && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, i}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -i}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{4, "E0", 0, 0, []int{}, []int{i, i, k, k}, "queued", 0, 0}}
	case i == k && j == l && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{4, "E0", 0, 0, []int{}, []int{i, i, j, j}, "queued", 0, 0}}
	case i == l && j == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{i, i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-i, -i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, []int{-j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{4, "E0", 0, 0, []int{}, []int{i, i, j, j}, "queued", 0, 0}}
	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []Job{
			Job{1, HashName(), 0, 0, []int{i, j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, -j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, -j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, -j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, -j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{i, j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, -j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Derivative(dims ...int) []Job {
	switch len(dims) {
	case 2:
		return Make2D(dims[0], dims[1])
	case 3:
		return Make3D(dims[0], dims[1], dims[2])
	case 4:
		return Make4D(dims[0], dims[1], dims[2], dims[3])
	}
	return []Job{Job{}}
}

type Job struct {
	Coeff   float64
	Name    string
	Number  int
	Count   int
	Steps   []int
	Index   []int
	Status  string
	Retries int
	Result  float64
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

func QueueAndWait(job Job, names []string, coords []float64, wg *sync.WaitGroup,
	ch chan int, totalJobs int, dump *GarbageHeap, fcs2 [][]float64,
	fcs3, fcs4 []float64, E0 float64, E2D [][]float64) {

	defer wg.Done()
	switch {
	case job.Name == "E0":
		job.Status = "done"
		job.Result = E0
		// add case for seconds in fourth if can hold seconds array
		// actually dont check index length because I want to use on seconds as well
		// case len(job.Index) == 4 && len(job.Step) == 2:
		// x,y are sorted, absolute value steps
		// if E2D[x][y] != 0 {
		// job.Status = "done"
		// job.Result = E2D[x][y]
		// } else {goto default}
	case len(job.Steps) == 2:
		x := IntAbs(job.Steps[0]) - 1
		y := IntAbs(job.Steps[1]) - 1
		if x > y {
			temp := x
			x = y
			y = temp
		}
		if E2D[x][y] != 0 {
			job.Status = "done"
			job.Result = E2D[x][y]
			break
		} 
		fallthrough
	default:
		coords = Step(coords, job.Steps...)
		molprofile := "inp/" + job.Name + ".inp"
		pbsfile := "inp/" + job.Name + ".pbs"
		outfile := "inp/" + job.Name + ".out"
		WriteMolproIn(molprofile, names, coords)
		WritePBS(pbsfile, molprofile, job.Count, dump)
		job.Number = Qsubmit(pbsfile)
		energy, err := ReadMolproOut(outfile)
		for err != nil {
			sigChan := make(chan os.Signal, 1)
			sigWant := os.Signal(syscall.Signal(job.Count))
			signal.Notify(sigChan, sigWant)
			select {
			// either receive signal
			case <-sigChan:
			// or timeout after 1 minute and retry
			case <-time.After(timeBeforeRetry):
			}
			energy, err = ReadMolproOut(outfile)
			if err != nil {
				Qsubmit(pbsfile)
			}
		}
		if err != nil {
			panic(err)
		}
		job.Status = "done"
		job.Result = energy
		dump.Heap = append(dump.Heap, "inp/"+Basename(molprofile))
	}
	switch len(job.Index) {
	case 2:
		x := job.Index[0] - 1
		y := job.Index[1] - 1
		fcs2[x][y] += job.Coeff * job.Result
	case 3:
		sort.Ints(job.Index)
		x := job.Index[0]
		y := job.Index[1]
		z := job.Index[2]
		// from spectro manual, subtract 1 for zero-indexed slice
		// reverse x, y, z because first dimension has to change slowest
		// x <= y <= z
		index := x + (y-1)*y/2 + (z-1)*z*(z+1)/6 - 1
		// fmt.Fprintln(os.Stderr, x, y, z, index)
		fcs3[index] += job.Coeff * job.Result
	case 4:
		sort.Ints(job.Index)
		x := job.Index[0]
		y := job.Index[1]
		z := job.Index[2]
		w := job.Index[3]
		index := x + (y-1)*y/2 + (z-1)*z*(z+1)/6 + (w-1)*w*(w+1)*(w+2)/24 - 1
		fcs4[index] += job.Coeff * job.Result
	}
	fmt.Fprintf(os.Stderr, "%d/%d jobs completed (%.1f%%)\n", progress, totalJobs,
		100*float64(progress)/float64(totalJobs))
	progress++
	<-ch
}

func RefEnergy(names []string, coords []float64, wg *sync.WaitGroup, c chan float64, dump *GarbageHeap) {
	defer wg.Done()
	molprofile := "inp/ref.inp"
	pbsfile := "inp/ref.pbs"
	outfile := "inp/ref.out"
	WriteMolproIn(molprofile, names, coords)
	WritePBS(pbsfile, molprofile, 35, dump)
	Qsubmit(pbsfile)
	energy, err := ReadMolproOut(outfile)
	for err != nil {
		time.Sleep(time.Second)
		energy, err = ReadMolproOut(outfile)
	}
	dump.Heap = append(dump.Heap, "inp/"+Basename(molprofile))
	c <- energy
}

func PrintFile15(fcs [][]float64, natoms int) int {
	fmt.Printf("%5d%5d\n", natoms, 6*natoms) // still not sure why this is just times 6
	flat := make([]float64, 0)
	for _, v := range fcs {
		flat = append(flat, v...)
	}
	for i := 0; i < len(flat); i += 3 {
		fmt.Printf("%20.10f%20.10f%20.10f\n", flat[i], flat[i+1], flat[i+2])
	}
	return len(flat)
}

func PrintFile30(fcs []float64, natoms, other int) {
	fmt.Printf("%5d%5d\n", natoms, other)
	for i := 0; i < len(fcs); i += 3 {
		fmt.Printf("%20.10f%20.10f%20.10f\n", fcs[i], fcs[i+1], fcs[i+2])
	}
}

func PrintFile40(fcs []float64, natoms, other int) {
	fmt.Printf("%5d%5d\n", natoms, other)
	for i := 0; i < len(fcs); i += 3 {
		fmt.Printf("%20.10f%20.10f%20.10f\n", fcs[i], fcs[i+1], fcs[i+2])
	}
}

func IntAbs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

func Drain(jobs []Job, names []string, coords []float64, wg *sync.WaitGroup,
	ch chan int, totalJobs int, dump *GarbageHeap, fcs2 [][]float64,
	fcs3, fcs4 []float64, E0 float64, count int, E2D [][]float64) {

	for job, _ := range jobs {
		wg.Add(1)
		ch <- 1
		jobs[job].Count = count
		if count == RTMAX {
			count = RTMIN
		} else {
			count++
		}
		go QueueAndWait(jobs[job], names, coords, wg, ch, totalJobs,
			dump, fcs2, fcs3, fcs4, E0, E2D)
	}
}

func main() {

	var (
		concRoutines int = 5
		nDerivative  int = 4
		names        []string
		coords       []float64
		ncoords      int
		wg           sync.WaitGroup
		dump         GarbageHeap
	)

	switch len(os.Args) {
	case 1:
		panic("Input geometry not found in command line args")
	case 2:
		geomfile := os.Args[1]
		names, coords = ReadInputXYZ(geomfile)
		ncoords = len(coords)
	case 3:
		geomfile := os.Args[1]
		names, coords = ReadInputXYZ(geomfile)
		ncoords = len(coords)
		concRoutines, _ = strconv.Atoi(os.Args[2])
	case 4:
		geomfile := os.Args[1]
		names, coords = ReadInputXYZ(geomfile)
		ncoords = len(coords)
		concRoutines, _ = strconv.Atoi(os.Args[2])
		nDerivative, _ = strconv.Atoi(os.Args[3])
	}

	if _, err := os.Stat("inp/"); os.IsNotExist(err) {
		os.Mkdir("inp", 0755)
	} else {
		os.RemoveAll("inp/")
		os.Mkdir("inp", 0755)
	}

	// run reference job
	c := make(chan float64)
	wg.Add(1)
	go RefEnergy(names, coords, &wg, c, &dump)
	E0 := <-c
	wg.Wait()
	close(c)

	// refactor so jobs are built and submitted at the same time
	// jobGroup := BuildJobList(names, coords, nDerivative)
	// func BuildJobList(names []string, coords []float64, nd int) (joblist []Job) {

	fcs2 := make([][]float64, ncoords)
	E2D := make([][]float64, ncoords)
	for i := 0; i < ncoords; i++ {
		fcs2[i] = make([]float64, ncoords)
		E2D[i] = make([]float64, ncoords)
	}
	natoms := len(names)
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	fcs3 := make([]float64, other3)
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	fcs4 := make([]float64, other4)

	ch := make(chan int, concRoutines)
	count := RTMIN // SIGRTMIN
	switch nDerivative {
	case 2:
		// 3 jobs for diagonal + 4 jobs for off diagonal
		totalJobs := ncoords*3 + (ncoords*ncoords-ncoords)*4
		for i := 1; i <= ncoords; i++ {
			for j := 1; j <= ncoords; j++ {
				jobs := Derivative(i, j)
				Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
			}
		}
	case 3:
		// third derivatives cap out at k <= j <= i
		// need to add third derivative factor here instead of 1140
		// totalJobs := ncoords*3 + (ncoords*ncoords-ncoords)*4 + 1140
		totalJobs := 1455
		for i := 1; i <= ncoords; i++ {
			for j := 1; j <= ncoords; j++ {
				jobs := Derivative(i, j)
				Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
				if j <= i {
					for k := 1; k <= j; k++ {
						jobs := Derivative(i, j, k)
						Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
					}
				}
			}
		}
	case 4:
		// add fourth derivative factor instead of 5000
		// totalJobs := ncoords*3 + (ncoords*ncoords-ncoords)*4 + 1140 + 5000
		totalJobs := 7440
		for i := 1; i <= ncoords; i++ {
			for j := 1; j <= ncoords; j++ {
				jobs := Derivative(i, j)
				Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
				if j <= i {
					for k := 1; k <= j; k++ {
						jobs := Derivative(i, j, k)
						Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
						for l := 1; l <= k; l++ {
							jobs := Derivative(i, j, k, l)
							Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, fcs2, fcs3, fcs4, E0, count, E2D)
						}
					}
				}
			}
		}
	}

	wg.Wait()

	// Unit conversion and denominator handling
	for i := 0; i < ncoords; i++ {
		for j := 0; j < ncoords; j++ {
			fcs2[i][j] = fcs2[i][j] * angborh * angborh / (4 * delta * delta)
		}
	}
	for i, _ := range fcs3 {
		fcs3[i] = fcs3[i] * angborh * angborh * angborh / (8 * delta * delta * delta)
	}
	for i, _ := range fcs4 {
		fcs4[i] = fcs4[i] * angborh * angborh * angborh * angborh / (16 * delta * delta * delta * delta)
	}

	switch nDerivative {
	case 2:
		PrintFile15(fcs2, natoms)
	case 3:
		PrintFile15(fcs2, natoms)
		PrintFile30(fcs3, natoms, other3)
	case 4:
		PrintFile15(fcs2, natoms)
		PrintFile30(fcs3, natoms, other3)
		PrintFile40(fcs4, natoms, other4)
	}
}
