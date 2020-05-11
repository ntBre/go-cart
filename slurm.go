package main

import (
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

type Slurm struct{}

func (s Slurm) MakeHead() []string {
	return []string{
		"#!/bin/bash",
		"#SBATCH --job-name=go-cart",
		"#SBATCH --ntasks=4",
		"#SBATCH --cpus-per-task=1",
		"#SBATCH -o /dev/null",
		"#SBATCH --mem=1gb"}
}

func (s Slurm) MakeFoot(count int, dump *GarbageHeap) []string {
	num := strconv.Itoa(count)
	return []string{"ssh -t master pkill -" + num + " " + progName,
		strings.Join(dump.Dump(), "\n")}
}

func (s Slurm) Make(filename string, count int, dump *GarbageHeap) []string {
	body := []string{"/home/qc/bin/molpro2018.sh 4 1 " + filename}
	return MakeInput(s.MakeHead(), s.MakeFoot(count, dump), body)
}

func (s Slurm) Write(pbsfile, molprofile string, count int, dump *GarbageHeap) {
	lines := s.Make(molprofile, count, dump)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(pbsfile, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func (s Slurm) Submit(filename string) int {
	out, _ := exec.Command("sbatch", filename).Output()
	// fmt.Println(string(out), err)
	// for err != nil {
	// 	time.Sleep(time.Second)
	// 	out, err = exec.Command("sbatch", filename).Output()
	// }
	b := Basename(string(out))
	i, _ := strconv.Atoi(b)
	return i
}
