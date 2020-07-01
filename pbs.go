package main

import (
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PBS implements the Submission interface
type PBS struct{}

// MakeHead makes a header for a PBS input file
func (p PBS) MakeHead() []string {
	return []string{"#!/bin/sh",
		"#PBS -N go-cart",
		"#PBS -S /bin/bash",
		"#PBS -j oe",
		"#PBS -o /dev/null",
		"#PBS -W umask=022",
		"#PBS -l walltime=100:00:00",
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
		"date"}
}

// MakeFoot makes a footer for a PBS input file
func (p PBS) MakeFoot(Sig1 int, dump *GarbageHeap) []string {
	sig1 := strconv.Itoa(Sig1)
	return []string{"ssh -t maple pkill -" + sig1 + " " + progName,
		strings.Join(dump.Dump(), "\n"), "rm -rf $TMPDIR"}
}

// Make calls MakeHead and MakeFoot to generate a PBS input file
func (p PBS) Make(filename string, Sig1 int, dump *GarbageHeap) []string {
	body := []string{"molpro -t 1 " + filename}
	return MakeInput(p.MakeHead(), p.MakeFoot(Sig1, dump), body)
}

// Write uses Make to write the contents of a PBS input file to
// filename
func (p PBS) Write(pbsfile, molprofile string, Sig1 int, dump *GarbageHeap) {
	lines := p.Make(molprofile, Sig1, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

// Submit executes the qsub command on filename
func (p PBS) Submit(filename string) int {
	// -f option to run qsub in foreground
	out, err := exec.Command("qsub", "-f", filename).Output()
	for err != nil {
		time.Sleep(time.Second)
		// just now adding -f to this one
		out, err = exec.Command("qsub", "-f", filename).Output()
	}
	b := Basename(string(out))
	i, _ := strconv.Atoi(b)
	return i
}
