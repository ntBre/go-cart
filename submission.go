package main

type Submission interface {
	MakeHead() []string
	MakeFoot(count int, dump *GarbageHeap) []string
	Make(filename string, count int, dump *GarbageHeap) []string
	Write(pbsfile, molprofile string, count int, dump *GarbageHeap)
	Submit(filename string) int
}
