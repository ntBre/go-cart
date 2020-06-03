package main

import (
	"math"
	"os"
	"reflect"
	"testing"
)

func TestMakeMolproIn(t *testing.T) {
	want := []string{
		"memory,1125,m",
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
		"set,spin=0",
		"hf",
		"{CCSD(T)-F12}"}
	got := TestProg.MakeIn(testnames, testcoords)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwanted %#v\n", got, want)
	}
}

func TestWriteMolproIn(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.in"
	TestProg.WriteIn(filename, testnames, testcoords)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("%s does not exist", filename)
	}
}

func TestReadMolproOut(t *testing.T) {
	// TODO write tests for all potential errors
	t.Run("Output file found and energy therein", func(t *testing.T) {
		filename := "testfiles/molpro.out"
		got, err := TestProg.ReadOut(filename)
		want := -76.369839607972
		if err != nil {
			t.Errorf("got an error, but didn't want one")
		} else if got != want {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})

	t.Run("Output file found but no energy therein", func(t *testing.T) {
		filename := "testfiles/brokenmolpro.out"
		got, err := TestProg.ReadOut(filename)
		want := brokenFloat
		if err == nil {
			t.Errorf("wanted an error, but didn't get one")
		} else if !math.IsNaN(got) {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})

	t.Run("No output file found", func(t *testing.T) {
		filename := "testfiles/molpro1.out"
		got, err := TestProg.ReadOut(filename)
		want := brokenFloat
		if err == nil {
			t.Errorf("wanted an error, but didn't get one")
		} else if !math.IsNaN(got) {
			t.Errorf("got %f, wanted %f", got, want)
		}
	})
}
