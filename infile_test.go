package main

import (
	"reflect"
	"testing"
)

func TestParseInfile(t *testing.T) {
	got := ParseInfile("sample.in")
	want := map[Key]string{
		GeomKey: "3\n" + "Comment\n" +
			"H          0.0000000000        0.7574590974        0.5217905143\n" +
			"O          0.0000000000        0.0000000000       -0.0657441568\n" +
			"H          0.0000000000       -0.7574590974        0.5217905143",
		ConcJobKey:     "9",
		DLevelKey:      "2",
		QueueTypeKey:   "SLURM",
		ChkIntervalKey: "120",
		ProgKey:        "MOPAC",
		DeltaKey: "0.010"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v\n", got, want)
	}
}
