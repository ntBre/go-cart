package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/maphash"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	energyLine       = "energy="
	molproTerminated = "Molpro calculation terminated"
	angborh          = 0.529177249
	progName         = "go-cart"
	RTMIN            = 35
	RTMAX            = 64
)

var (
	ErrEnergyNotFound          = errors.New("Energy not found in Molpro output")
	ErrFileNotFound            = errors.New("Molpro output file not found")
	ErrEnergyNotParsed         = errors.New("Energy not parsed in Molpro output")
	ErrFinishedButNoEnergy     = errors.New("Molpro output finished but no energy found")
	ErrFileContainsError       = errors.New("Molpro output file contains an error")
	concRoutines           int = 5
	workers                int = 0
	delta                      = 0.005
	progress                   = 1
	Sig1                       = RTMIN
	brokenFloat                = math.NaN()
	// may need to adjust this if jobs can reasonably take longer than a minute
	timeBeforeRetry            = time.Second * 60
	Q               Submission = PBS{}
	fc2Mutex        sync.Mutex
	fc3Mutex        sync.Mutex
	fc4Mutex        sync.Mutex
	e2dMutex        sync.Mutex
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

func MakeInput(head, foot, body []string) []string {
	file := make([]string, 0)
	file = append(file, head...)
	file = append(file, body...)
	file = append(file, foot...)
	return file
}

func Basename(filename string) string {
	file := path.Base(filename)
	re := regexp.MustCompile(path.Ext(file))
	basename := re.ReplaceAllString(file, "")
	return basename
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

type Job struct {
	Coeff   float64
	Name    string
	Number  int
	Sig1    int
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

func E2DIndex(n, ncoords int) int {
	if n < 0 {
		return IntAbs(n) + ncoords - 1
	}
	return n - 1
}
func QueueAndWait(job Job, names []string, coords []float64, wg *sync.WaitGroup,
	ch chan int, totalJobs int, dump *GarbageHeap, fcs2 [][]float64,
	fcs3, fcs4 []float64, E0 float64, E2D [][]float64) {

	defer wg.Done()
	switch {
	case job.Name == "E0":
		job.Status = "done"
		job.Result = E0
	case len(job.Steps) == 2:
		x := E2DIndex(job.Steps[0], len(coords))
		y := E2DIndex(job.Steps[1], len(coords))
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
		Q.Write(pbsfile, molprofile, job.Sig1, dump)
		job.Number = Q.Submit(pbsfile)
		energy, err := ReadMolproOut(outfile)
		for err != nil {
			sigChan := make(chan os.Signal, 1)
			sig1Want := os.Signal(syscall.Signal(job.Sig1))
			signal.Notify(sigChan, sig1Want)
			select {
			// either receive signal
			case <-sigChan:
				// or timeout after and retry
				fmt.Println("got signal ", sig1Want, " for step ", progress)
			case <-time.After(timeBeforeRetry):
				fmt.Println("didn't get signal, waiting on step ", progress)
			}
			energy, err = ReadMolproOut(outfile)
			if err != nil {
				fmt.Printf("error %s at step %d with %d workers\n", err, progress, workers)
				fmt.Println(outfile)
			}
			if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError) ||
				(err == ErrFileNotFound && workers < concRoutines/2) {
				fmt.Println("resubmitting for", err)
				Q.Submit(pbsfile)
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
	// Locks to prevent concurrent access to the same index
	case 2:
		if len(job.Steps) == 2 {
			E2Dx := E2DIndex(job.Steps[0], len(coords))
			E2Dy := E2DIndex(job.Steps[1], len(coords))
			if E2Dx > E2Dy {
				temp := E2Dx
				E2Dx = E2Dy
				E2Dy = temp
			}
			e2dMutex.Lock()
			E2D[E2Dx][E2Dy] = job.Result
			e2dMutex.Unlock()
		}
		x := job.Index[0] - 1
		y := job.Index[1] - 1
		fc2Mutex.Lock()
		fcs2[x][y] += job.Coeff * job.Result
		fc2Mutex.Unlock()
	case 3:
		sort.Ints(job.Index)
		x := job.Index[0]
		y := job.Index[1]
		z := job.Index[2]
		// from spectro manual, subtract 1 for zero-indexed slice
		// reverse x, y, z because first dimension has to change slowest
		// x <= y <= z
		index := x + (y-1)*y/2 + (z-1)*z*(z+1)/6 - 1
		fc3Mutex.Lock()
		fcs3[index] += job.Coeff * job.Result
		fc3Mutex.Unlock()
	case 4:
		sort.Ints(job.Index)
		x := job.Index[0]
		y := job.Index[1]
		z := job.Index[2]
		w := job.Index[3]
		index := x + (y-1)*y/2 + (z-1)*z*(z+1)/6 + (w-1)*w*(w+1)*(w+2)/24 - 1
		fc4Mutex.Lock()
		fcs4[index] += job.Coeff * job.Result
		fc4Mutex.Unlock()
	}
	fmt.Fprintf(os.Stderr, "%d/%d jobs completed (%.1f%%)\n", progress, totalJobs,
		100*float64(progress)/float64(totalJobs))
	progress++
	workers--
	<-ch
}

func RefEnergy(names []string, coords []float64, wg *sync.WaitGroup, c chan float64, dump *GarbageHeap) {
	defer wg.Done()
	molprofile := "inp/ref.inp"
	pbsfile := "inp/ref.pbs"
	outfile := "inp/ref.out"
	WriteMolproIn(molprofile, names, coords)
	Q.Write(pbsfile, molprofile, 35, dump)
	Q.Submit(pbsfile)
	energy, err := ReadMolproOut(outfile)
	for err != nil {
		time.Sleep(time.Second)
		energy, err = ReadMolproOut(outfile)
	}
	dump.Heap = append(dump.Heap, "inp/"+Basename(molprofile))
	c <- energy
}

func PrintFile15(fcs [][]float64, natoms int) int {
	f, _ := os.Create("fort.15")
	fmt.Fprintf(f, "%5d%5d\n", natoms, 6*natoms) // still not sure why this is just times 6
	flat := make([]float64, 0)
	for _, v := range fcs {
		flat = append(flat, v...)
	}
	for i := 0; i < len(flat); i += 3 {
		fmt.Fprintf(f, "%20.10f%20.10f%20.10f\n", flat[i], flat[i+1], flat[i+2])
	}
	return len(flat)
}

func PrintFile30(fcs []float64, natoms, other int) {
	f, _ := os.Create("fort.30")
	fmt.Fprintf(f, "%5d%5d\n", natoms, other)
	for i := 0; i < len(fcs); i += 3 {
		fmt.Fprintf(f, "%20.10f%20.10f%20.10f\n", fcs[i], fcs[i+1], fcs[i+2])
	}
}

func PrintFile40(fcs []float64, natoms, other int) {
	f, _ := os.Create("fort.40")
	fmt.Fprintf(f, "%5d%5d\n", natoms, other)
	for i := 0; i < len(fcs); i += 3 {
		fmt.Fprintf(f, "%20.10f%20.10f%20.10f\n", fcs[i], fcs[i+1], fcs[i+2])
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
	fcs3, fcs4 []float64, E0 float64, E2D [][]float64) {

	for job, _ := range jobs {
		wg.Add(1)
		workers++
		ch <- 1
		// this probably belongs in the job creation part
		jobs[job].Sig1 = Sig1
		// When they hit RTMAX roll over to RTMIN
		if Sig1 == RTMAX {
			Sig1 = RTMIN
		} else {
			Sig1++
		}
		go QueueAndWait(jobs[job], names, coords, wg, ch, totalJobs,
			dump, fcs2, fcs3, fcs4, E0, E2D)
	}
}

func TotalJobs(nd, ncoords int) (total int) {
	// this is a disgusting way to calculate this
	// 3 jobs for diagonal + 4 jobs for off diagonal
	// totalJobs := ncoords*3 + (ncoords*ncoords-ncoords)*4
	for i := 1; i <= ncoords; i++ {
		for j := 1; j <= ncoords; j++ {
			total += len(Derivative(i, j))
			if nd > 2 && j <= i {
				for k := 1; k <= j; k++ {
					total += len(Derivative(i, j, k))
					if nd > 3 {
						for l := 1; l <= k; l++ {
							total += len(Derivative(i, j, k, l))
						}
					}
				}
			}
		}
	}
	return
}

func MakeCheckpoint(fcs2 [][]float64, fcs3, fcs4 []float64, indices ...int) {
	fc2, _ := json.Marshal(fcs2)
	ioutil.WriteFile("fc2.json", fc2, 0755)
	fc3, _ := json.Marshal(fcs3)
	ioutil.WriteFile("fc3.json", fc3, 0755)
	fc4, _ := json.Marshal(fcs4)
	ioutil.WriteFile("fc4.json", fc4, 0755)
	index := ""
	for _, i := range indices {
		index += fmt.Sprintf("%d ", i)
	}
	id, _ := json.Marshal(index)
	ioutil.WriteFile("id.json", id, 0755)
}

func main() {

	var (
		nDerivative int = 4
		checkAfter  int = 100
		names       []string
		coords      []float64
		ncoords     int
		wg          sync.WaitGroup
		dump        GarbageHeap
	)

	// BEGIN ParseOSArgs()
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
	case 5:
		geomfile := os.Args[1]
		names, coords = ReadInputXYZ(geomfile)
		ncoords = len(coords)
		concRoutines, _ = strconv.Atoi(os.Args[2])
		nDerivative, _ = strconv.Atoi(os.Args[3])
		if os.Args[4] == strconv.Itoa(1) {
			Q = Slurm{}
			// timeBeforeRetry = 200 * time.Second
		}
	case 6:
		geomfile := os.Args[1]
		names, coords = ReadInputXYZ(geomfile)
		ncoords = len(coords)
		concRoutines, _ = strconv.Atoi(os.Args[2])
		nDerivative, _ = strconv.Atoi(os.Args[3])
		if os.Args[4] == strconv.Itoa(1) {
			Q = Slurm{}
			// timeBeforeRetry = 200 * time.Second
		}
		checkAfter, _ = strconv.Atoi(os.Args[5])
	}
	// END ParseOSArgs()

	if _, err := os.Stat("inp/"); os.IsNotExist(err) {
		os.Mkdir("inp", 0755)
	} else {
		os.RemoveAll("inp/")
		os.Mkdir("inp", 0755)
	}

	// run reference job
	// I think I can eliminate the channels and waitgroup here
	// this is blocking anyway as it's set up
	c := make(chan float64)
	wg.Add(1)
	go RefEnergy(names, coords, &wg, c, &dump)
	E0 := <-c
	wg.Wait()
	close(c)

	// BEGIN InitFCArrays
	fcs2 := make([][]float64, ncoords)
	E2D := make([][]float64, 2*ncoords)
	for i := 0; i < 2*ncoords; i++ {
		if i < ncoords {
			fcs2[i] = make([]float64, ncoords)
		}
		E2D[i] = make([]float64, 2*ncoords)
	}

	natoms := len(names)
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	fcs3 := make([]float64, other3)
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	fcs4 := make([]float64, other4)
	// END InitFCArrays

	ch := make(chan int, concRoutines)

	totalJobs := TotalJobs(nDerivative, ncoords)
	var i, j, k, l int
	for i = 1; i <= ncoords; i++ {
		for j = 1; j <= ncoords; j++ {
			if progress%checkAfter == 0 {
				MakeCheckpoint(fcs2, fcs3, fcs4, i, j, k, l)
			}
			jobs := Derivative(i, j)
			Drain(jobs, names, coords, &wg, ch, totalJobs, &dump,
				fcs2, fcs3, fcs4, E0, E2D)
			if nDerivative > 2 && j <= i {
				for k = 1; k <= j; k++ {
					if progress%checkAfter == 0 {
						MakeCheckpoint(fcs2, fcs3, fcs4, i, j, k, l)
					}
					jobs := Derivative(i, j, k)
					Drain(jobs, names, coords, &wg, ch, totalJobs, &dump,
						fcs2, fcs3, fcs4, E0, E2D)
					if nDerivative > 3 {
						for l = 1; l <= k; l++ {
							if progress%checkAfter == 0 {
								MakeCheckpoint(fcs2, fcs3, fcs4, i, j, k, l)
							}
							jobs := Derivative(i, j, k, l)
							Drain(jobs, names, coords, &wg, ch, totalJobs, &dump,
								fcs2, fcs3, fcs4, E0, E2D)
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

	PrintFile15(fcs2, natoms)
	if nDerivative > 2 {
		PrintFile30(fcs3, natoms, other3)
	}
	if nDerivative > 3 {
		PrintFile40(fcs4, natoms, other4)
	}

}
