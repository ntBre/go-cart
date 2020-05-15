package main

import (
	"reflect"
	"testing"
)

var (
	m Program = Mopac{}
)

func TestMakeHead(t *testing.T) {
	got := m.MakeHead()
	want := []string{
		"threads=1 XYZ ANGSTROMS scfcrt=1.D-21 aux(precision=9) " +
			"external=params.dat 1SCF charge=0 PM6", "MOLECULE # 1", "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestMakeIn(t *testing.T) {
	want := []string{
		"threads=1 XYZ ANGSTROMS scfcrt=1.D-21 aux(precision=9) " +
			"external=params.dat 1SCF charge=0 PM6", "MOLECULE # 1", "",
		"H 0.0000000000 0.7574590974 0.5217905143",
		"O 0.0000000000 0.0000000000 -0.0657441568",
		"H 0.0000000000 -0.7574590974 0.5217905143",
	}
	got := m.MakeIn(testnames, testcoords)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
