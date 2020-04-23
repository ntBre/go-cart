package main

import (
	"os"
	"reflect"
	"testing"
	"math"
)

var testnames = []string{"H", "O", "H"}
var testcoords = []float64{0.0000000000, 0.7574590974, 0.5217905143,
	0.0000000000, 0.0000000000, -0.0657441568,
	0.0000000000, -0.7574590974, 0.5217905143}

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
		"date",
		"rm -rf $TMPDIR"}
	got := MakePBS(filename)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v", got, want)
	}
}

func TestWritePBS(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.pbs"
	WritePBS(filename, "molpro.in")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("%s does not exist", filename)
	}
}

func TestQsubmit(t *testing.T) {
	filename := "testfiles/molpro.in"
	got := Qsubmit(filename)
	want := 775241
	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}
}

func TestDerivative(t *testing.T) {
	t.Run("Diagonal second derivative", func(t *testing.T) {
		got := Derivative(1, 1)
		want := "E(+1+1) - 2*E(0) + E(-1-1) / (2d)^2"
		if got != want {
			t.Errorf("got %s, wanted %s", got, want)
		}
	})
	t.Run("Off-diagonal second derivative", func(t *testing.T) {
		got := Derivative(1, 2)
		want := "E(+1+2) - E(+1-2) - E(-1+2) + E(-1-2) / (2d)^2"
		if got != want {
			t.Errorf("got %s, wanted %s", got, want)
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
		got := Step(testnames, testcoords, 1, 1)
		want := []float64{1.0000000000, 0.7574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Negative two steps", func(t *testing.T) {
		got := Step(testnames, testcoords, -1, -1)
		want := []float64{-1.0000000000, 0.7574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Plus-minus two step", func(t *testing.T) {
		got := Step(testnames, testcoords, +1, -2)
		want := []float64{0.5000000000, 0.2574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !approxeq(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
	t.Run("Minus-plus two step", func(t *testing.T) {
		got := Step(testnames, testcoords, -1, 2)
		want := []float64{-0.5000000000, 1.2574590974, 0.5217905143,
			0.0000000000, 0.0000000000, -0.0657441568,
			0.0000000000, -0.7574590974, 0.5217905143}
		if !approxeq(got, want) {
			t.Errorf("got %#v, wanted %#v", got, want)
		}
	})
}
