package main

import (
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type PBS struct{}
// implements Submission

func (p PBS) MakeHead() []string {
	return []string{"#!/bin/sh",
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
		"date"}
}

func (p PBS) MakeFoot(Sig1 int, dump *GarbageHeap) []string {
	sig1 := strconv.Itoa(Sig1)
	return []string{"ssh -t maple pkill -" + sig1 + " " + progName,
		strings.Join(dump.Dump(), "\n"), "rm -rf $TMPDIR"}
}

func (p PBS) Make(filename string, Sig1 int, dump *GarbageHeap) []string {
	body := []string{"molpro -t 1 " + filename}
	return MakeInput(p.MakeHead(), p.MakeFoot(Sig1, dump), body)
}

func (p PBS) Write(pbsfile, molprofile string, Sig1 int, dump *GarbageHeap) {
	lines := p.Make(molprofile, Sig1, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

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
