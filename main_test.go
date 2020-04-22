package main

import (
	"os"
	"reflect"
	"testing"
)

var testgeom = Geometry{[]string{"H", "O", "H"},
	[]float64{0.0000000000, 0.7574590974, 0.5217905143,
		0.0000000000, 0.0000000000, -0.0657441568,
		0.0000000000, -0.7574590974, 0.5217905143}}

func TestReadInputXYZ(t *testing.T) {
	want := testgeom
	got := ReadInputXYZ("testfiles/geom.xyz")
	if !reflect.DeepEqual(got, want) {
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
	got := MakeMolproIn(&testgeom)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwanted %#v\n", got, want)
	}
}

func TestWriteMolproIn(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.in"
	WriteMolproIn(filename, &testgeom)
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
