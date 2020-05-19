package main

import (
	"os"
	"reflect"
	"testing"
	"strconv"
)

func TestMakePBSFoot(t *testing.T) {
	num := strconv.Itoa(5)
	want := []string{"ssh -t maple pkill -" + num + " " + "go-cart",
		"rm test1*\nrm test2*\nrm test3*",
		"rm -rf $TMPDIR"}
	tdump := GarbageHeap{Heap: []string{"test1", "test2", "test3"}}
	got := P.MakeFoot(5, &tdump)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v", got, want)
	}
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
		"#PBS -l mem=9gb",
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
		"rm test1*\nrm test2*\nrm test3*",
		"rm -rf $TMPDIR"}
	tdump := GarbageHeap{Heap: []string{"test1", "test2", "test3"}}
	got := P.Make(filename, 35, &tdump)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v", got, want)
	}
}

func TestWritePBS(t *testing.T) {
	// this is a terrible test after it has been run once successfully
	filename := "testfiles/molpro.pbs"
	tdump := GarbageHeap{Heap: []string{"test1", "test2", "test3"}}
	P.Write(filename, "molpro.in", 35, &tdump)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("%s does not exist", filename)
	}
}

func TestQsubmit(t *testing.T) {
	filename := "testfiles/molpro.pbs"
	got := P.Submit(filename)
	want := 775241
	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}
}
