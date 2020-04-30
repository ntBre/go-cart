package main

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"
)

var testnames = []string{"H", "O", "H"}
var testcoords = []float64{0.0000000000, 0.7574590974, 0.5217905143,
	0.0000000000, 0.0000000000, -0.0657441568,
	0.0000000000, -0.7574590974, 0.5217905143}
var minname = []string{"H"}
var mincoord = []float64{0.0000000000, 0.7574590974, 0.5217905143}

func TestReadInputXYZ(t *testing.T) {
	want, want2 := testnames, testcoords
	got, got2 := ReadInputXYZ("testfiles/geom.xyz")
	if !reflect.DeepEqual(got, want) && !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMakeMolproIn(t *testing.T) {
	want := []string{
		"memory,50,m",
		"nocompress",
		"geomtyp=xyz",
		"angstrom",
		"geometry={",
		"H 0.0000000000 0.7574590974 0.5217905143",
		"O 0.0000000000 0.0000000000 -0.0657441568",
		"H 0.0000000000 -0.7574590974 0.5217905143",
		"}",
		"basis=cc-pVTZ-F12",
		"set,charge=0",
		"set,spin",
		"hf",
		"{CCSD(T)-F12}"}
	got := MakeMolproIn(testnames, testcoords)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwanted %#v\n", got, want)
	}
}

func TestWriteMolproIn(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.in"
	WriteMolproIn(filename, testnames, testcoords)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("%s does not exist", filename)
	}
}

func TestReadMolproOut(t *testing.T) {
	t.Run("Output file found and energy therein", func(t *testing.T) {
		filename := "testfiles/molpro.out"
		got, err := ReadMolproOut(filename)
		want := -76.369839607972
		if err != nil {
			t.Errorf("got an error, but didn't want one")
		} else if got != want {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})

	t.Run("Output file found but no energy therein", func(t *testing.T) {
		filename := "testfiles/brokenmolpro.out"
		got, err := ReadMolproOut(filename)
		want := brokenFloat
		if err == nil {
			t.Errorf("wanted an error, but didn't get one")
		} else if got != want {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})

	t.Run("No output file found", func(t *testing.T) {
		filename := "testfiles/molpro1.out"
		got, err := ReadMolproOut(filename)
		want := brokenFloat
		if err == nil {
			t.Errorf("wanted an error, but didn't get one")
		} else if got != want {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})
}

func TestMakePBS(t *testing.T) {
	filename := "molpro.in"
	want := []string{
		"#!/bin/sh",
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
		"date",
		"molpro -t 1 molpro.in",
		"ssh -t maple pkill -35 go-cart",
		"rm -rf $TMPDIR"}
	got := MakePBS(filename, 35)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v", got, want)
	}
}

func TestWritePBS(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.pbs"
	WritePBS(filename, "molpro.in", 35)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("%s does not exist", filename)
	}
}

// func TestQsubmit(t *testing.T) {
// 	filename := "testfiles/molpro.pbs"
// 	got := Qsubmit(filename)
// 	want := 775241
// 	if got != want {
// 		t.Errorf("got %d, wanted %d", got, want)
// 	}
// }

func TestDerivative(t *testing.T) {
	t.Run("Diagonal second derivative", func(t *testing.T) {
		got := Derivative(1, 1)[0]
		want := Job{1, "untestedName", 0, 0, []int{1, 1}, "queued", 0, 0}
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
		want := Job{1, "untestedName", 0, 0, []int{1, 2}, "queued", 0, 0}
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

func TestStep(t *testing.T) {
	approxeq := func(a, b []float64) (eq bool) {
		eps := 1e-6
		eq = true
		for i, _ := range a {
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

func TestBuildJobList(t *testing.T) {
	// fmt.Println(len(BuildJobList(minname, mincoord)))
	// fmt.Println(BuildJobList(minname, mincoord))
}

// func TestPrintFile15(t *testing.T) {
// 	testinp := [][]float64{{1.157599172074697e-08, 0, -9.947598300641403e-13, -1.09899929157109e-08, 0, 0, -9.6801500149013e-10, 0, 0},
// 		{0, 0.00013374501598661936, 8.392485599983956e-05, 0, -0.00012260568999522548, -7.274317201222402e-05, 0, -1.1139001003357407e-05, -1.118235799424383e-05},
// 		{-9.947598300641403e-13, 8.392485599983956e-05, 7.791444299698469e-05, -9.947598300641403e-13, -9.510723299399615e-05, -8.207080801980737e-05, 0, 1.118235799424383e-05, 4.1567160025124394e-06},
// 		{-1.09899929157109e-08, 0, -9.947598300641403e-13, 2.274600774399005e-08, -2.0179413695586845e-12, 0, -1.09899929157109e-08, 0, 0},
// 		{0, -0.00012260568999522548, -9.510723299399615e-05, -2.0179413695586845e-12, 0.00024521072199945593, 0, 0, -0.00012260568999522548, 9.510723299399615e-05},
// 		{0, -7.274317201222402e-05, -8.207080801980737e-05, 0, 0, 0.00016414090299576856, 0, 7.274317201222402e-05, -8.2070806001866e-05},
// 		{-9.6801500149013e-10, 0, 0, -1.09899929157109e-08, 0, 0, 1.157599172074697e-08, 0, 9.947598300641403e-13},
// 		{0, -1.1139001003357407e-05, 1.118235799424383e-05, 0, -0.00012260568999522548, 7.274317201222402e-05, 0, 0.0001337450169813792, -8.392485599983956e-05},
// 		{0, -1.118235799424383e-05, 4.1567160025124394e-06, 0, 9.510723299399615e-05, -8.2070806001866e-05, 9.947598300641403e-13, -8.392485599983956e-05, 7.791444299698469e-05}}
// 	PrintFile15(testinp)
// }

func TestCentral(t *testing.T) {
	t.Run("Diagonal second derivative", func(t *testing.T) {
		want := "+1f(2Dx)-2f(0Dx)+1f(-2Dx)"
		got := Central(2, "x")
		if got != want {
			t.Errorf("got %s, wanted %s\n", got, want)
		}
	})
	t.Run("Diagonal third derivative", func(t *testing.T) {
		want := "+1f(3Dx)-3f(1Dx)+3f(-1Dx)-1f(-3Dx)"
		got := Central(3, "x")
		if got != want {
			t.Errorf("got %s, wanted %s\n", got, want)
		}
	})
	t.Run("Diagonal fourth derivative", func(t *testing.T) {
		want := "+1f(4Dx)-4f(2Dx)+6f(0Dx)-4f(-2Dx)+1f(-4Dx)"
		got := Central(4, "x")
		if got != want {
			t.Errorf("got %s, wanted %s\n", got, want)
		}
	})
}
