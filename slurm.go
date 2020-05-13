package main

import (
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Slurm struct{}

func (s Slurm) MakeHead() []string {
	return []string{
		"#!/bin/bash",
		"#SBATCH --job-name=go-cart",
		"#SBATCH --ntasks=1",
		"#SBATCH --cpus-per-task=1",
		"#SBATCH -o /dev/null",
		"#SBATCH --no-requeue",
		"#SBATCH --mem=9gb"}
}

func (s Slurm) MakeFoot(Sig1, Sig2 int, dump *GarbageHeap) []string {
	sig1 := strconv.Itoa(Sig1)
	sig2 := strconv.Itoa(Sig2)
	return []string{"ssh -t master pkill -" + sig1 + " " + progName,
		"ssh -t master pkill -" + sig2 + " " + progName,
		strings.Join(dump.Dump(), "\n")}
}

func (s Slurm) Make(filename string, Sig1, Sig2 int, dump *GarbageHeap) []string {
	body := []string{"/home/qc/bin/molpro2018.sh 1 1 " + filename}
	return MakeInput(s.MakeHead(), s.MakeFoot(Sig1, Sig2, dump), body)
}

func (s Slurm) Write(pbsfile, molprofile string, Sig1, Sig2 int, dump *GarbageHeap) {
	lines := s.Make(molprofile, Sig1, Sig2, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func (s Slurm) Submit(filename string) int {
	out, err := exec.Command("sbatch", filename).Output()
	// have to use sbatch because srun grabs a whole node
	// and runs interactively
	for err != nil {
		time.Sleep(time.Second)
		out, err = exec.Command("sbatch", filename).Output()
	}
	b := Basename(string(out))
	i, _ := strconv.Atoi(b)
	return i
}
