package main

// Submission is an interface for queueing systems
type Submission interface {
	MakeHead() []string
	MakeFoot(Sig1 int, dump *GarbageHeap) []string
	Make(filename string, Sig1 int, dump *GarbageHeap) []string
	Write(pbsfile, molprofile string, Sig1 int, dump *GarbageHeap)
	Submit(filename string) int
}
