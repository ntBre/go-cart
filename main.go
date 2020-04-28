package main

import (
	"errors"
	"fmt"
	"hash/maphash"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	energyLine  = "energy="
	brokenFloat = 999.999
	angborh     = 0.529177249
)

var (
	ErrEnergyNotFound = errors.New("Energy not found in Molpro output")
	ErrFileNotFound   = errors.New("Molpro output file not found")
	delta             = 0.005
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
	runtime.LockOSThread()
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
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
			// dont delete, too many syscalls
			// files, _ := filepath.Glob("inp/" + Basename(filename) + "*")
			// for _, f := range files {
			// 	os.Remove(f)
			// }
			runtime.UnlockOSThread()
			return f, nil
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

func Qsubmit(filename string) {

	runtime.LockOSThread()
	_, err := exec.Command("qsub", filename).Output()
	runtime.UnlockOSThread()
	retries := 0
	for err != nil {
		if retries < 5 {
			runtime.LockOSThread()
			time.Sleep(time.Second)
			_, err = exec.Command("qsub", filename).Output()
			runtime.UnlockOSThread()
			fmt.Println(err)
			retries++
		} else {
			panic(fmt.Sprintf("Qsubmit failed after %d retries", retries))
		}
	}
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
		return []Job{Job{1, HashName(), []int{i, i}, "queued", 0, 0},
			Job{-2, "E0", []int{0}, "queued", 0, 0},
			Job{1, HashName(), []int{-i, -i}, "queued", 0, 0}}
	} else {
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []Job{Job{1, HashName(), []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), []int{i, -j}, "queued", 0, 0},
			Job{-1, HashName(), []int{-i, j}, "queued", 0, 0},
			Job{1, HashName(), []int{-i, -j}, "queued", 0, 0}}
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
	Coeff   float64
	Name    string
	Steps   []int // doubles as index in array
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

func BuildJobList(names []string, coords []float64) (joblist [][]Job) {
	for i := 1; i <= len(coords); i++ {
		// should be j <= i, but just do all for now before taking into account symmetry
		for j := 1; j <= len(coords); j++ {
			joblist = append(joblist, Derivative(i, j))
		}
	}
	return
}

func QueueAndWait(job *Job, names []string, coords []float64, wg *sync.WaitGroup) {
	defer wg.Done()
	coords = Step(coords, job.Steps...)
	molprofile := "inp/" + job.Name + ".inp"
	pbsfile := "inp/" + job.Name + ".pbs"
	outfile := "inp/" + job.Name + ".out"
	WriteMolproIn(molprofile, names, coords)
	WritePBS(pbsfile, molprofile)
	Qsubmit(pbsfile)
	energy, err := ReadMolproOut(outfile)
	for err != nil {
		time.Sleep(time.Second)
		energy, err = ReadMolproOut(outfile)
		job.Retries++
	}
	job.Status = "done"
	job.Result = energy
}

func RefEnergy(names []string, coords []float64, wg *sync.WaitGroup, c chan float64) {
	defer wg.Done()
	molprofile := "inp/ref.inp"
	pbsfile := "inp/ref.pbs"
	outfile := "inp/ref.out"
	WriteMolproIn(molprofile, names, coords)
	WritePBS(pbsfile, molprofile)
	Qsubmit(pbsfile)
	energy, err := ReadMolproOut(outfile)
	for err != nil {
		time.Sleep(time.Second)
		energy, err = ReadMolproOut(outfile)
	}
	c <- energy
}

func PrintFile15(fcs [][]float64) {
	flat := make([]float64, 0)
	for _, v := range fcs {
		flat = append(flat, v...)
	}
	for i := 0; i < len(flat); i += 3 {
		fmt.Printf("%20.10f%20.10f%20.10f\n", flat[i], flat[i+1], flat[i+2])
	}

}

func main() {
	if len(os.Args) < 2 {
		panic("Input geometry not found in command line args")
	}
	geomfile := os.Args[1]
	names, coords := ReadInputXYZ(geomfile)
	fcs := make([][]float64, len(coords))

	if _, err := os.Stat("inp/"); os.IsNotExist(err) {
		os.Mkdir("inp", 0755)
	} else {
		os.RemoveAll("inp/")
		os.Mkdir("inp", 0755)
	}

	var wg, wgOuter sync.WaitGroup
	c := make(chan float64)
	wg.Add(1)
	go RefEnergy(names, coords, &wg, c)
	E0 := <-c
	wg.Wait()
	close(c)

	jobGroups := BuildJobList(names, coords)

	for i, _ := range jobGroups {
		fcs[i/len(coords)] = make([]float64, len(coords))
	}

	ch := make(chan int, 2)
	for i, jobGroup := range jobGroups {
		var wg sync.WaitGroup
		// trying locking threads when reading output
		// channel method isnt running the last job - never ended with without passing wgOuter
		// trying passing wgOuter
		ch <- 1 // try moving this inside the closure
		wgOuter.Add(1)
		go func(wg sync.WaitGroup, wgOuter *sync.WaitGroup, i int, jobGroup []Job) {
			for j, _ := range jobGroup {
				if jobGroup[j].Name != "E0" {
					wg.Add(1)
					go QueueAndWait(&jobGroup[j], names, coords, &wg)
				} else {
					jobGroup[j].Status = "done"
					jobGroup[j].Result = E0
				}
			}
			wg.Wait()
			var total float64 = 0
			for j, _ := range jobGroup {
				total += jobGroup[j].Coeff * jobGroup[j].Result
			}
			x := jobGroup[0].Steps[0] - 1
			y := jobGroup[0].Steps[1] - 1
			// hard coded second derivative scaling factor and denominator
			fcs[x][y] = total * angborh * angborh / (4 * delta * delta)
			fmt.Fprintf(os.Stderr, "%d/%d jobs completed\n", i+1, len(jobGroups))
			<-ch
			wgOuter.Done() // move to bottom, scared of defer
			// trying again with pointer since it still wasnt ending
		}(wg, &wgOuter, i, jobGroup)
	}
	wgOuter.Wait()
	PrintFile15(fcs)
}
