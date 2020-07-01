package main

import (
	"fmt"
	"math"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var (
	P          = PBS{}
	TestProg   = Molpro{}
	testnames  = []string{"H", "O", "H"}
	testcoords = []float64{0.0000000000, 0.7574590974, 0.5217905143,
		0.0000000000, 0.0000000000, -0.0657441568,
		0.0000000000, -0.7574590974, 0.5217905143}
	minname  = []string{"H"}
	mincoord = []float64{0.0000000000, 0.7574590974, 0.5217905143}
	testinp  = [][]float64{
		{1.157599172074697e-08, 0, -9.947598300641403e-13, -1.09899929157109e-08, 0, 0, -9.6801500149013e-10, 0, 0},
		{0, 0.00013374501598661936, 8.392485599983956e-05, 0, -0.00012260568999522548, -7.274317201222402e-05, 0,
			-1.1139001003357407e-05, -1.118235799424383e-05},
		{-9.947598300641403e-13, 8.392485599983956e-05, 7.791444299698469e-05, -9.947598300641403e-13,
			-9.510723299399615e-05, -8.207080801980737e-05, 0, 1.118235799424383e-05, 4.1567160025124394e-06},
		{-1.09899929157109e-08, 0, -9.947598300641403e-13, 2.274600774399005e-08, -2.0179413695586845e-12, 0,
			-1.09899929157109e-08, 0, 0},
		{0, -0.00012260568999522548, -9.510723299399615e-05, -2.0179413695586845e-12, 0.00024521072199945593, 0, 0,
			-0.00012260568999522548, 9.510723299399615e-05},
		{0, -7.274317201222402e-05, -8.207080801980737e-05, 0, 0, 0.00016414090299576856, 0, 7.274317201222402e-05,
			-8.2070806001866e-05},
		{-9.6801500149013e-10, 0, 0, -1.09899929157109e-08, 0, 0, 1.157599172074697e-08, 0, 9.947598300641403e-13},
		{0, -1.1139001003357407e-05, 1.118235799424383e-05, 0, -0.00012260568999522548, 7.274317201222402e-05, 0,
			0.0001337450169813792, -8.392485599983956e-05},
		{0, -1.118235799424383e-05, 4.1567160025124394e-06, 0, 9.510723299399615e-05, -8.2070806001866e-05,
			9.947598300641403e-13, -8.392485599983956e-05, 7.791444299698469e-05}}
	testinp30 = [][][]float64{{{1, 2, 3}, {4, 5, 6}},
		{{7, 8, 9}, {10, 11, 12}}}
	testinp40 = [][][][]float64{
		{
			{
				{1, 2, 3},
				{4, 5, 6},
			},
			{
				{7, 8, 9},
				{10, 11, 12},
			},
		},
		{
			{
				{13, 14, 15},
				{16, 17, 18},
			},
			{
				{19, 20, 21},
				{22, 23, 24},
			},
		},
	}
)

func TestReadFile(t *testing.T) {
	got, _ := ReadFile("testfiles/geom.xyz")
	want := []string{
		"3",
		"Comment",
		"H          0.0000000000        0.7574590974        0.5217905143",
		"O          0.0000000000        0.0000000000       -0.0657441568",
		"H          0.0000000000       -0.7574590974        0.5217905143"}
	if !reflect.DeepEqual(got, want) {
		fmt.Println(cmp.Diff(got, want))
		t.Errorf("got %#v, wanted %#v\n", got, want)
	}
}

func TestSplitLine(t *testing.T) {
	got := strings.Fields("H          0.0000000000        0.7574590974        0.5217905143")
	want := []string{"H", "0.0000000000", "0.7574590974", "0.5217905143"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s, wanted %v\n", got, want)
	}
}

func TestReadInputXYZ(t *testing.T) {
	want, want2 := testnames, testcoords
	lines, _ := ReadFile("testfiles/geom.xyz")
	got, got2 := ReadInputXYZ(lines)
	if !reflect.DeepEqual(got, want) && !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMakeInput(t *testing.T) {
	got := MakeInput([]string{"1", "2", "3"},
		[]string{"7", "8", "9"},
		[]string{"4", "5", "6"})
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestBasename(t *testing.T) {
	got := Basename("/home/brent/Projects/go-cart/go-cart.exe")
	want := "go-cart"
	if got != want {
		t.Errorf("got %s, wanted %s\n", got, want)
	}
}

func TestDump(t *testing.T) {
	tdump := GarbageHeap{Heap: []string{"test1", "test2", "test3"}}
	got := tdump.Dump()
	want := []string{"rm test1*", "rm test2*", "rm test3*"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestStep(t *testing.T) {
	approxeq := func(a, b []float64) (eq bool) {
		eps := 1e-6
		eq = true
		for i := range a {
			if math.Abs(a[i]-b[i]) > eps {
				eq = false
				return
			}
		}
		return
	}
	t.Run("Positive two steps", func(t *testing.T) {
		got := Step(testcoords, 1, 1)
		want := []float64{2 * delta, 0.7574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Negative two steps", func(t *testing.T) {
		got := Step(testcoords, -1, -1)
		want := []float64{-2 * delta, 0.7574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Plus-minus two step", func(t *testing.T) {
		got := Step(testcoords, +1, -2)
		want := []float64{delta, 0.7574590974 - delta, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !approxeq(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Minus-plus two step", func(t *testing.T) {
		got := Step(testcoords, -1, 2)
		want := []float64{-delta, 0.7574590974 + delta, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !approxeq(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
}

func TestHashName(t *testing.T) {
	got := HashName()
	if got[:3] != "job" {
		t.Errorf("got %s, wanted %s", got[:3], "job")
	}
}

func TestE2dIndex(t *testing.T) {
	t.Run("positive number", func(t *testing.T) {
		got := E2dIndex(2, 9)
		want := 1
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
	t.Run("negative number", func(t *testing.T) {
		got := E2dIndex(-2, 9)
		want := 10
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
}

func TestIndex3(t *testing.T) {
	got := Index3(9, 9, 9)
	want := 164
	if got != want {
		t.Errorf("got %d, wanted %d\n", got, want)
	}
}

func TestIndex4(t *testing.T) {
	got := Index4(9, 9, 9, 9)
	want := 494
	if got != want {
		t.Errorf("got %d, wanted %d\n", got, want)
	}
}

func TestHandleSignal(t *testing.T) {
	t.Run("received signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 5*time.Second)
			c <- err
		}(c)
		exec.Command("pkill", "-35", "go-cart").Run()
		err := <-c
		if err != nil {
			t.Errorf("did not receive signal")
		}
	})
	t.Run("no signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 50*time.Millisecond)
			c <- err
		}(c)
		exec.Command("pkill", "-34", "go-cart").Run()
		err := <-c
		if err == nil {
			t.Errorf("received signal and didn't want one")
		}
	})
}

func TestDerivative(t *testing.T) {
	t.Run("Diagonal second derivative", func(t *testing.T) {
		got := Derivative(1, 1)[0]
		want := Job{1, "untestedName", 0, 0, []int{1, 1}, []int{1, 1}, "queued", 0, 0}
		if want.Coeff != got.Coeff ||
			!reflect.DeepEqual(want.Steps, got.Steps) ||
			want.Status != got.Status ||
			want.Retries != got.Retries ||
			want.Result != got.Result {
			fmt.Println(want.Steps, got.Steps)
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Off-diagonal second derivative", func(t *testing.T) {
		got := Derivative(1, 2)[0]
		want := Job{1, "untestedName", 0, 0, []int{1, 2}, []int{1, 2}, "queued", 0, 0}
		if want.Coeff != got.Coeff ||
			!reflect.DeepEqual(want.Steps, got.Steps) ||
			want.Status != got.Status ||
			want.Retries != got.Retries ||
			want.Result != got.Result {
			fmt.Println(want.Steps, got.Steps)
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
}

func TestPrintFile15(t *testing.T) {
	fc2test := [][]float64{
		[]float64{1, 2, 3},
		[]float64{4, 5, 6},
	}
	got := PrintFile15(fc2test, 3, "testfiles/fort.15")
	want := 6
	if got != want {
		t.Errorf("not the right length, watch out")
	}
}

func TestPrintFile30(t *testing.T) {
	fc3test := []float64{1, 2, 3, 4, 5, 6}
	got := PrintFile30(fc3test, 3, 165, "testfiles/fort.30")
	want := 6
	if got != want {
		t.Errorf("not the right length, watch out")
	}
}

func TestPrintFile40(t *testing.T) {
	fc4test := []float64{1, 2, 3, 4, 5, 6}
	got := PrintFile40(fc4test, 3, 495, "testfiles/fort.40")
	want := 6
	if got != want {
		t.Errorf("not the right length, watch out")
	}
}

func TestIntAbs(t *testing.T) {
	t.Run("negative number", func(t *testing.T) {
		got := IntAbs(-2)
		want := 2
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
	t.Run("positive number", func(t *testing.T) {
		got := IntAbs(2)
		want := 2
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
	t.Run("zero", func(t *testing.T) {
		got := IntAbs(0)
		want := 0
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
}

func TestTotalJobs(t *testing.T) {
	t.Run("2nd derivative, water", func(t *testing.T) {
		got := TotalJobs(2, 9)
		want := 315
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
	t.Run("3rd derivative, water", func(t *testing.T) {
		got := TotalJobs(3, 9)
		want := 1455
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
	t.Run("4th derivative, water", func(t *testing.T) {
		got := TotalJobs(4, 9)
		want := 7440
		if got != want {
			t.Errorf("got %d, wanted %d\n", got, want)
		}
	})
}

// TODO Make/ReadCheckpoint

func TestSetParams(t *testing.T) {
	wantBefore := concRoutines == 5 && nDerivative == 4 && Queue == PBS{} &&
		checkAfter == 100 && Prog == Molpro{} && delta == 0.005
	if !wantBefore {
		t.Errorf("something wrong to start")
	}
	SetParams("sample.in")
	wantAfter := concRoutines == 9 && nDerivative == 2 && Queue == Slurm{} &&
		checkAfter == 120 && Prog == Mopac{} && delta == 0.010
	if !wantAfter {
		t.Error("something wrong after SetParams", concRoutines, nDerivative, Queue, checkAfter, Prog, delta)
	}
}

// TODO InitFCArrays
