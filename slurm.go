package main

import (
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Slurm implements the Submission interface for the Slurm queueing
// system
type Slurm struct{}

// MakeHead returns the header for a Slurm input file
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

// MakeFoot returns the footer for a Slurm input file
func (s Slurm) MakeFoot(Sig1 int, dump *GarbageHeap) []string {
	sig1 := strconv.Itoa(Sig1)
	return []string{"ssh -t master pkill -" + sig1 + " " + progName,
		strings.Join(dump.Dump(), "\n")}
}

// Make uses MakeHead and MakeFoot to return the contents of a Slurm
// input file
func (s Slurm) Make(filename string, Sig1 int, dump *GarbageHeap) []string {
	body := []string{"/home/qc/bin/molpro2018.sh 1 1 " + filename}
	return MakeInput(s.MakeHead(), s.MakeFoot(Sig1, dump), body)
}

// Write uses Make to write the contents of a Slurm input file to
// filename
func (s Slurm) Write(pbsfile, molprofile string, Sig1 int, dump *GarbageHeap) {
	lines := s.Make(molprofile, Sig1, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

// Submit runs the sbatch command on filename
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
