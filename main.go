package main

import (
	"errors"
	"fmt"
	"hash/maphash"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	energyLine  = "energy="
	brokenFloat = 999.999
	angborh     = 0.529177249
	maxRetries  = 5
	progName    = "go-cart"
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

func Qsubmit(filename string) int {

	runtime.LockOSThread()
	out, err := exec.Command("qsub", filename).Output()
	runtime.UnlockOSThread()
	retries := 0
	for err != nil {
		if retries < maxRetries {
			runtime.LockOSThread()
			time.Sleep(time.Second)
			out, err = exec.Command("qsub", filename).Output()
			runtime.UnlockOSThread()
			fmt.Println(err)
			retries++
		} else {
			panic(fmt.Sprintf("Qsubmit failed after %d retries", retries))
		}
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
		// "#PBS -o /dev/null",
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

func MakePBSFoot(count int) []string {
	num := strconv.Itoa(count)
	return []string{"echo signaling " + num,
		"ssh -t maple pkill -" + num + " " + progName,
		"rm -rf $TMPDIR"}
}

func MakePBS(filename string, count int) []string {
	body := []string{"molpro -t 1 " + filename}
	return MakeInput(MakePBSHead(), MakePBSFoot(count), body)
}

func WritePBS(pbsfile, molprofile string, count int) {
	lines := MakePBS(molprofile, count)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func Make2D(i, j int) []Job {
	if i == j {
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []Job{Job{1, HashName(), 0, 0, []int{i, i}, "queued", 0, 0},
			Job{-2, "E0", 0, 0, []int{i, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -i}, "queued", 0, 0}}
	} else {
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []Job{Job{1, HashName(), 0, 0, []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{i, -j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, []int{-i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, []int{-i, -j}, "queued", 0, 0}}
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
	Number  int
	Count   int
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

func BuildJobList(names []string, coords []float64) (joblist []Job) {
	for i := 1; i <= len(coords); i++ {
		for j := 1; j <= len(coords); j++ {
			joblist = append(joblist, Derivative(i, j)...)
		}
	}
	return
}

func Qstat(jobnum string) string {
	qstatus, _ := exec.Command("qstat", jobnum).Output()
	qlines := strings.Split(string(qstatus), "\n")
	if len(qlines) == 4 {
		return SplitLine(qlines[2])[4]
	}
	return "done"
}

func QueueAndWait(job *Job, names []string, coords []float64, wg *sync.WaitGroup, ch chan int) {
	defer wg.Done()
	coords = Step(coords, job.Steps...)
	molprofile := "inp/" + job.Name + ".inp"
	pbsfile := "inp/" + job.Name + ".pbs"
	outfile := "inp/" + job.Name + ".out"
	WriteMolproIn(molprofile, names, coords)
	WritePBS(pbsfile, molprofile, job.Count)
	job.Number = Qsubmit(pbsfile)
	// this is the place to check for queue status if energy not found
	// failing when file exists but not written to
	// if file exists and there's no energy=/pattern match AND not in queue, resubmit
	// jobnum := strconv.Itoa(job.Number)
	energy, err := ReadMolproOut(outfile)
	sigChan := make(chan os.Signal, 1)
	sigWant := os.Signal(syscall.Signal(job.Count))
	signal.Notify(sigChan, sigWant)
	fmt.Println("want signal", sigWant)
	s := <-sigChan
	fmt.Println("Got signal", s)
	energy, err = ReadMolproOut(outfile)
	job.Retries++
	if job.Retries > 10 {
		fmt.Fprintf(os.Stderr, "having problems with %s\n", outfile)
		job.Number = Qsubmit(pbsfile)
		// jobnum = strconv.Itoa(job.Number)
		job.Retries = 0
	}
	if err != nil {
		panic(err)
	}
	job.Status = "done"
	job.Result = energy
	<-ch
}

func RefEnergy(names []string, coords []float64, wg *sync.WaitGroup, c chan float64) {
	defer wg.Done()
	molprofile := "inp/ref.inp"
	pbsfile := "inp/ref.pbs"
	outfile := "inp/ref.out"
	WriteMolproIn(molprofile, names, coords)
	WritePBS(pbsfile, molprofile, 35)
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

func IntAbs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

func main() {
	if len(os.Args) < 2 {
		panic("Input geometry not found in command line args")
	}
	geomfile := os.Args[1]
	names, coords := ReadInputXYZ(geomfile)
	ncoords := len(coords)

	var concRoutines int
	if len(os.Args) > 2 {
		concRoutines, _ = strconv.Atoi(os.Args[2])
	} else {
		concRoutines = 5
	}

	if _, err := os.Stat("inp/"); os.IsNotExist(err) {
		os.Mkdir("inp", 0755)
	} else {
		os.RemoveAll("inp/")
		os.Mkdir("inp", 0755)
	}

	var wg sync.WaitGroup
	c := make(chan float64)
	wg.Add(1)
	go RefEnergy(names, coords, &wg, c)
	E0 := <-c
	wg.Wait()
	close(c)

	jobGroup := BuildJobList(names, coords)

	fcs2 := make([][]float64, ncoords)
	fcs3 := make([][][]float64, ncoords)
	fcs4 := make([][][][]float64, ncoords)
	for i := 0; i < ncoords; i++ {
		fcs2[i] = make([]float64, ncoords)
		fcs3[i] = make([][]float64, len(coords))
		fcs4[i] = make([][][]float64, len(coords))
		for j := 0; j < ncoords; j++ {
			fcs3[i][j] = make([]float64, len(coords))
			fcs4[i][j] = make([][]float64, len(coords))
			for k := 0; k < ncoords; k++ {
				fcs4[i][j][k] = make([]float64, len(coords))
			}
		}
	}

	ch := make(chan int, concRoutines)
	count := 34 // SIGRTMIN
	for j, _ := range jobGroup {
		if jobGroup[j].Name != "E0" {
			wg.Add(1)
			ch <- 1
			jobGroup[j].Count = count
			if count == 64 {
				count = 34
			} else {
				count++
			}
			fmt.Println(jobGroup[j].Count)
			go QueueAndWait(&jobGroup[j], names, coords, &wg, ch)
		} else {
			jobGroup[j].Status = "done"
			jobGroup[j].Result = E0
		}
	}
	wg.Wait()
	for j, _ := range jobGroup {
		switch len(jobGroup[j].Steps) {
		case 2:
			x := jobGroup[j].Steps[0]
			y := jobGroup[j].Steps[1]
			x = IntAbs(x) - 1
			y = IntAbs(y) - 1
			fcs2[x][y] += jobGroup[j].Coeff * jobGroup[j].Result
		case 3:
			x := jobGroup[j].Steps[0]
			y := jobGroup[j].Steps[1]
			z := jobGroup[j].Steps[2]
			x = IntAbs(x) - 1
			y = IntAbs(y) - 1
			z = IntAbs(z) - 1
			fcs3[x][y][z] += jobGroup[j].Coeff * jobGroup[j].Result
		case 4:
			x := jobGroup[j].Steps[0]
			y := jobGroup[j].Steps[1]
			z := jobGroup[j].Steps[2]
			w := jobGroup[j].Steps[3]
			x = IntAbs(x) - 1
			y = IntAbs(y) - 1
			z = IntAbs(z) - 1
			w = IntAbs(w) - 1
			fcs4[x][y][z][w] += jobGroup[j].Coeff * jobGroup[j].Result
		}
	}
	// hard coded second derivative scaling factor and denominator
	for i := 0; i < ncoords; i++ {
		for j := 0; j < ncoords; j++ {
			fcs2[i][j] = fcs2[i][j] * angborh * angborh / (4 * delta * delta)
		}
	}
	PrintFile15(fcs2)
}
