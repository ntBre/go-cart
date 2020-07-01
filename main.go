package main

import (
	"encoding/json"
	"errors"
	"flag"
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
	molproTerminated = "Molpro calculation terminated"
	angbohr          = 0.529177249
	progName         = "go-cart"
	RTMIN            = 35
	RTMAX            = 64
)

var (
	fc2Scale = angbohr * angbohr / (4 * delta * delta)
	fc3Scale = angbohr * angbohr * angbohr / (8 * delta * delta * delta)
	fc4Scale = angbohr * angbohr * angbohr * angbohr / (16 * delta * delta * delta * delta)
)

// Error messages
var (
	ErrEnergyNotFound      = errors.New("Energy not found in Molpro output")
	ErrFileNotFound        = errors.New("Molpro output file not found")
	ErrEnergyNotParsed     = errors.New("Energy not parsed in Molpro output")
	ErrFinishedButNoEnergy = errors.New("Molpro output finished but no energy found")
	ErrFileContainsError   = errors.New("Molpro output file contains an error")
	ErrBlankOutput         = errors.New("Molpro output file exists but is blank")
	ErrInputGeomNotFound   = errors.New("Geometry not found in input file")
	ErrTimeout             = errors.New("Timeout waiting for signal")
)

// Input parameters with default values
var (
	concRoutines int        = 5
	nDerivative  int        = 4
	Queue        Submission = PBS{}
	checkAfter   int        = 100
	Prog         Program    = Molpro{}
	delta        float64    = 0.005
	molproMethod string     = "CCSD(T)-F12"
	mopacMethod  string     = "PM6"
	basis        string     = "cc-pVTZ-F12"
	charge       string     = "0"
	spin         string     = "0"
	energyLine              = "energy="
)

// Shared variables
// use RWMutex instead of Mutex because concurrent reads are okay
var (
	progress            = 1
	Sig1                = RTMIN
	brokenFloat         = math.NaN()
	timeBeforeRetry     = time.Second * 15
	workers         int = 0
	fc2Mutex        sync.RWMutex
	fc3Mutex        sync.RWMutex
	fc4Mutex        sync.RWMutex
	e2dMutex        sync.RWMutex
	fc2CountMutex   sync.RWMutex
	fc3CountMutex   sync.RWMutex
	fc4CountMutex   sync.RWMutex
	fc2             [][]float64
	fc3             []float64
	fc4             []float64
	e2d             [][]float64
	fc2Done         [][]float64
	fc3Done         []float64
	fc4Done         []float64
	e2dDone         [][]float64
	fc2Count        [][]int
	fc3Count        []int
	fc4Count        []int
)

// Command line flags
var (
	checkpoint = flag.Bool("c", false, "resume from checkpoint")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
)

const (
	help = `Requires:
- gocart input file, with a geometry in Angstroms
Flags:
`
)

func ReadFile(filename string) ([]string, error) {
	lines, err := ioutil.ReadFile(filename)
	// trim trailing newlines
	split := strings.Split(strings.TrimSpace(string(lines)), "\n")
	for i, line := range split {
		split[i] = strings.TrimSpace(line)
	}
	return split, err
}

func SplitLine(line string) []string {
	re := regexp.MustCompile(`\s+`)
	trim := strings.TrimSpace(line)
	s := strings.Split(strings.TrimSpace(re.ReplaceAllString(trim, " ")), " ")
	return s
}

func ReadInputXYZ(split []string) ([]string, []float64) {
	// split, _ := ReadFile(filename)
	names := make([]string, 0)
	coords := make([]float64, 0)
	// skip the natoms and comment line in xyz file
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
	Status  string // not used
	Retries int    // not used
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

func E2dIndex(n, ncoords int) int {
	if n < 0 {
		return IntAbs(n) + ncoords - 1
	}
	return n - 1
}

func Index3(x, y, z int) int {
	return x + (y-1)*y/2 + (z-1)*z*(z+1)/6 - 1
}

func Index4(x, y, z, w int) int {
	return x + (y-1)*y/2 + (z-1)*z*(z+1)/6 + (w-1)*w*(w+1)*(w+2)/24 - 1
}

func HandleSignal(sig int, timeout time.Duration) error {
	sigChan := make(chan os.Signal, 1)
	// trying to stop goroutine leak
	// stop receiving signals to send on channel
	// then close the channel
	defer signal.Stop(sigChan)
	defer close(sigChan)
	sig1Want := os.Signal(syscall.Signal(sig))
	signal.Notify(sigChan, sig1Want)
	select {
	// either receive signal
	case <-sigChan:
		fmt.Println("got signal ", sig1Want, " for step ", progress)
		return nil
	// or timeout after and retry
	case <-time.After(timeout):
		fmt.Println("didn't get signal, waiting on step ", progress)
		return ErrTimeout
	}
}

func QueueAndWait(job Job, names []string, coords []float64, wg *sync.WaitGroup,
	ch chan int, totalJobs int, dump *GarbageHeap, E0 float64) {

	defer wg.Done()
	switch {
	case job.Name == "E0":
		job.Status = "done"
		job.Result = E0
	case len(job.Steps) == 2:
		x := E2dIndex(job.Steps[0], len(coords))
		y := E2dIndex(job.Steps[1], len(coords))
		if x > y {
			temp := x
			x = y
			y = temp
		}
		if e2d[x][y] != 0 {
			job.Status = "done"
			job.Result = e2d[x][y]
			break
		}
		fallthrough
	default:
		coords = Step(coords, job.Steps...)
		molprofile := "inp/" + job.Name + ".inp"
		pbsfile := "inp/" + job.Name + ".pbs"
		outfile := "inp/" + job.Name + ".out"
		Prog.WriteIn(molprofile, names, coords)
		Queue.Write(pbsfile, molprofile, job.Sig1, dump)
		job.Number = Queue.Submit(pbsfile)
		energy, err := Prog.ReadOut(outfile)
		for err != nil {
			HandleSignal(job.Sig1, timeBeforeRetry)
			energy, err = Prog.ReadOut(outfile)
			if err != nil {
				fmt.Printf("error %s at step %d with %d workers\n",
					err, progress, workers)
				fmt.Println(outfile)
			}
			if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError || err == ErrBlankOutput) ||
				(err == ErrFileNotFound && workers < concRoutines/2) {
				fmt.Println("resubmitting for", err)
				Queue.Submit(pbsfile)
			}
		}
		if err != nil {
			panic(err)
		}
		job.Status = "done"
		job.Result = energy
		dump.Heap = append(dump.Heap, "inp/"+Basename(molprofile))
	}
	// TODO should test something in here/DRY it up
	// looks repetitive but not immediately clear how to fix
	switch len(job.Index) {
	// Locks to prevent concurrent access to the same index
	// fcDone doesn't need a lock because it's only written once per index
	case 2:
		if len(job.Steps) == 2 {
			e2dx := E2dIndex(job.Steps[0], len(coords))
			e2dy := E2dIndex(job.Steps[1], len(coords))
			if e2dx > e2dy {
				temp := e2dx
				e2dx = e2dy
				e2dy = temp
			}
			e2dMutex.Lock()
			e2d[e2dx][e2dy] = job.Result
			e2dMutex.Unlock()
		}
		x := job.Index[0] - 1
		y := job.Index[1] - 1
		fc2Mutex.Lock()
		fc2[x][y] += job.Coeff * job.Result
		fc2Mutex.Unlock()
		fc2CountMutex.Lock()
		fc2Count[x][y]--
		fc2CountMutex.Unlock()
		if fc2Count[x][y] == 0 {
			fc2Done[x][y] = fc2[x][y]
		}
	case 3:
		sort.Ints(job.Index) // need to be in order, from spectro manual
		index := Index3(job.Index[0], job.Index[1], job.Index[2])
		fc3Mutex.Lock()
		fc3[index] += job.Coeff * job.Result
		fc3Mutex.Unlock()
		fc3CountMutex.Lock()
		fc3Count[index]--
		fc3CountMutex.Unlock()
		if fc3Count[index] == 0 {
			fc3Done[index] = fc3[index]
		}
	case 4:
		sort.Ints(job.Index)
		index := Index4(job.Index[0], job.Index[1], job.Index[2], job.Index[3])
		fc4Mutex.Lock()
		fc4[index] += job.Coeff * job.Result
		fc4Mutex.Unlock()
		fc4CountMutex.Lock()
		fc4Count[index]--
		fc4CountMutex.Unlock()
		if fc4Count[index] == 0 {
			fc4Done[index] = fc4[index]
		}
	}
	fmt.Fprintf(os.Stderr, "%d/%d jobs completed (%.1f%%)\n", progress, totalJobs,
		100*float64(progress)/float64(totalJobs))
	progress++
	if checkAfter > 0 && progress%checkAfter == 0 {
		MakeCheckpoint()
	}
	workers--
	<-ch
}

func RefEnergy(names []string, coords []float64, dump *GarbageHeap) (energy float64) {
	molprofile := "inp/ref.inp"
	pbsfile := "inp/ref.pbs"
	outfile := "inp/ref.out"
	Prog.WriteIn(molprofile, names, coords)
	Queue.Write(pbsfile, molprofile, 35, dump)
	Queue.Submit(pbsfile)
	energy, err := Prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Second)
		energy, err = Prog.ReadOut(outfile)
	}
	dump.Heap = append(dump.Heap, "inp/"+Basename(molprofile))
	return
}

func PrintFile15(fc [][]float64, natoms int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, 6*natoms) // still not sure why this is just times 6
	flat := make([]float64, 0)
	for _, v := range fc {
		flat = append(flat, v...)
	}
	for i := range flat {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", flat[i]*fc2Scale)
	}
	return len(flat)
}

func PrintFile30(fc []float64, natoms, other int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, other)
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i]*fc3Scale)
	}
	return len(fc)
}

func PrintFile40(fc []float64, natoms, other int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, other)
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i]*fc4Scale)
	}
	return len(fc)
}

func IntAbs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

func Drain(jobs []Job, names []string, coords []float64, wg *sync.WaitGroup,
	ch chan int, totalJobs int, dump *GarbageHeap, E0 float64) {

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
		go QueueAndWait(jobs[job], names, coords, wg, ch, totalJobs, dump, E0)
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

func MakeCheckpoint() {
	fc2Json, _ := json.Marshal(fc2Done)
	ioutil.WriteFile("fc2.json", fc2Json, 0755)
	fc3Json, _ := json.Marshal(fc3Done)
	ioutil.WriteFile("fc3.json", fc3Json, 0755)
	fc4Json, _ := json.Marshal(fc4Done)
	ioutil.WriteFile("fc4.json", fc4Json, 0755)
	e2dJson, _ := json.Marshal(e2d)
	ioutil.WriteFile("e2d.json", e2dJson, 0755)
}

func ReadCheckpoint() {
	fc2lines, _ := ioutil.ReadFile("fc2.json")
	fc3lines, _ := ioutil.ReadFile("fc3.json")
	fc4lines, _ := ioutil.ReadFile("fc4.json")
	e2dlines, _ := ioutil.ReadFile("e2d.json")
	err := json.Unmarshal(fc2lines, &fc2)
	err = json.Unmarshal(fc3lines, &fc3)
	err = json.Unmarshal(fc4lines, &fc4)
	// also put back into *Done for check in main
	err = json.Unmarshal(fc2lines, &fc2Done)
	err = json.Unmarshal(fc3lines, &fc3Done)
	err = json.Unmarshal(fc4lines, &fc4Done)
	err = json.Unmarshal(e2dlines, &e2d)
	if err != nil {
		panic(err)
	}
}

func SetParams(filename string) (names []string, coords []float64, err error) {
	err = ErrInputGeomNotFound
	keymap := ParseInfile(filename)

	// defaults

	for key, value := range keymap {
		switch key {
		case ConcJobKey:
			concRoutines, err = strconv.Atoi(value)
		case DLevelKey:
			nDerivative, err = strconv.Atoi(value)
		case QueueTypeKey:
			switch value {
			case "PBS":
				Queue = PBS{}
			case "SLURM":
				Queue = Slurm{}
			}
		case ChkIntervalKey:
			checkAfter, err = strconv.Atoi(value)
		case ProgKey:
			switch value {
			case "MOPAC":
				Prog = Mopac{}
			case "MOLPRO":
				Prog = Molpro{}
			case "CCCR":
				Prog = CcCR{}
				energyLine = "SETTING CCCRE"
			}
		case GeomKey:
			lines := strings.Split(value, "\n")
			names, coords = ReadInputXYZ(lines)
		case DeltaKey:
			delta, err = strconv.ParseFloat(value, 64)
		case MethodKey:
			molproMethod = value
			mopacMethod = value
		case BasisKey:
			basis = value
		case ChargeKey:
			charge = value
		case SpinKey:
			spin = value
		}
	}
	return
}

// ParseFlags parses the command line flags and returns the remaining
// arguments.
func ParseFlags() []string {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), help)
		flag.PrintDefaults()
	}
	flag.Parse()
	return flag.Args()
}

/*
Instead of 3 arrays for each derivative level
maybe I want to have an array of structs with fields
Value float64, Count int
but then have to check them all when marshalling so idk
probably some other solution that is actually good
*/

func InitFCArrays(ncoords int) (int, int) {
	fc2 = make([][]float64, ncoords)
	fc2Done = make([][]float64, ncoords)
	fc2Count = make([][]int, ncoords)
	e2d = make([][]float64, 2*ncoords)
	for i := 0; i < 2*ncoords; i++ {
		if i < ncoords {
			fc2[i] = make([]float64, ncoords)
			fc2Done[i] = make([]float64, ncoords)
			fc2Count[i] = make([]int, ncoords)
		}
		e2d[i] = make([]float64, 2*ncoords)
	}
	natoms := ncoords / 3
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	fc3 = make([]float64, other3)
	fc3Done = make([]float64, other3)
	fc3Count = make([]int, other3)
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	fc4 = make([]float64, other4)
	fc4Done = make([]float64, other4)
	fc4Count = make([]int, other4)
	return other3, other4
}

func main() {

	var (
		names   []string
		coords  []float64
		ncoords int
		natoms  int
		wg      sync.WaitGroup
		dump    GarbageHeap
		err     error
	)

	Args := ParseFlags()

	switch len(Args) {
	case 0:
		panic("Input file not found in command line args")
	case 1:
		names, coords, err = SetParams(Args[0])
		ncoords = len(coords)
		natoms = len(names)
		if err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat("inp/"); os.IsNotExist(err) {
		os.Mkdir("inp", 0755)
	} else {
		if *overwrite {
			os.RemoveAll("inp/")
			os.Mkdir("inp", 0755)
		} else {
			panic("Directory inp already exists, overwrite with -o")
		}
	}

	other3, other4 := InitFCArrays(ncoords)

	if *checkpoint {
		ReadCheckpoint()
	}

	E0 := RefEnergy(names, coords, &dump)

	ch := make(chan int, concRoutines)

	totalJobs := TotalJobs(nDerivative, ncoords)
	for i := 1; i <= ncoords; i++ {
		for j := 1; j <= ncoords; j++ {
			if fc2Done[i-1][j-1] == 0 {
				jobs := Derivative(i, j)
				fc2Count[i-1][j-1] = len(jobs)
				Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, E0)
			}
			if nDerivative > 2 && j <= i {
				for k := 1; k <= j; k++ {
					// Index3/4 require arguments to be sorted
					temp := []int{i, j, k}
					sort.Ints(temp)
					index := Index3(temp[0], temp[1], temp[2])
					if fc3Done[index] == 0 {
						jobs := Derivative(i, j, k)
						fc3Count[index] = len(jobs)
						Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, E0)
					} // else should increment progress and print for checkpoints
					if nDerivative > 3 {
						for l := 1; l <= k; l++ {
							temp := []int{i, j, k, l}
							sort.Ints(temp)
							index := Index4(temp[0], temp[1], temp[2], temp[3])
							if fc4Done[index] == 0 {
								jobs := Derivative(i, j, k, l)
								fc4Count[index] = len(jobs)
								Drain(jobs, names, coords, &wg, ch, totalJobs, &dump, E0)
							}
						}
					}
				}
			}
		}
	}

	wg.Wait()

	PrintFile15(fc2, natoms, "fort.15")
	if nDerivative > 2 {
		PrintFile30(fc3, natoms, other3, "fort.30")
	}
	if nDerivative > 3 {
		PrintFile40(fc4, natoms, other4, "fort.40")
	}

}
