package main

func Make2D(i, j int) []Job {
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i}, []int{i, i}, "queued", 0, 0},
			Job{-2, "E0", 0, 0, 0, []int{}, []int{i, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i}, []int{i, i}, "queued", 0, 0}}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, j}, []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, -j}, []int{i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, j}, []int{i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -j}, []int{i, j}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Make3D(i, j, k int) []Job {
	switch {
	case i == j && i == k:
		// E(+i+i+i) - 3*E(i) + 3*E(-i) -E(-i-i-i) / (2d)^3
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, i}, []int{i, i, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{i}, []int{i, i, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{-i}, []int{i, i, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -i}, []int{i, i, i}, "queued", 0, 0}}
	case i == j && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, k}, []int{i, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{k}, []int{i, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, k}, []int{i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, -k}, []int{i, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-k}, []int{i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -k}, []int{i, i, k}, "queued", 0, 0}}
	case i == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, j}, []int{i, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{j}, []int{i, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, j}, []int{i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, -j}, []int{i, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-j}, []int{i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -j}, []int{i, i, j}, "queued", 0, 0}}
	case j == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{j, j, i}, []int{j, j, i}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i}, []int{j, j, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, i}, []int{j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, -i}, []int{j, j, i}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-i}, []int{j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, -i}, []int{j, j, i}, "queued", 0, 0}}
	case i != j && i != k && j != k:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, -j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -j, k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, -j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, j, -k}, []int{i, j, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -j, -k}, []int{i, j, k}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Make4D(i, j, k, l int) []Job {
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, i, i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{-4, HashName(), 0, 0, 0, []int{i, i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{6, "E0", 0, 0, 0, []int{}, []int{i, i, i, i}, "queued", 0, 0},
			Job{-4, HashName(), 0, 0, 0, []int{-i, -i}, []int{i, i, i, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -i, -i}, []int{i, i, i, i}, "queued", 0, 0}}
	// 3 and 1
	case i == j && i == k && i != l:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{-i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -i, l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{-i, -l}, []int{i, i, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -i, -l}, []int{i, i, i, l}, "queued", 0, 0}}
	case i == j && i == l && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{-i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -i, k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{-i, -k}, []int{i, i, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -i, -k}, []int{i, i, i, k}, "queued", 0, 0}}
	case i == k && i == l && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{-i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -i, j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{-i, -j}, []int{i, i, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -i, -j}, []int{i, i, i, j}, "queued", 0, 0}}
	case j == k && j == l && j != i:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{j, j, j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{-j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, -j, i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{3, HashName(), 0, 0, 0, []int{j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{-3, HashName(), 0, 0, 0, []int{-j, -i}, []int{j, j, j, i}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, -j, -i}, []int{j, j, j, i}, "queued", 0, 0}}
	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		// x -> i, y -> k, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, -k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -k, l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-k, -l}, []int{i, i, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -k, -l}, []int{i, i, k, l}, "queued", 0, 0}}
	case i == k && i != j && i != l && j != l:
		// x -> i, y -> j, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, -j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -j, l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-j, -l}, []int{i, i, j, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -j, -l}, []int{i, i, j, l}, "queued", 0, 0}}
	case i == l && i != j && i != k && j != k:
		// x -> i, y -> k, z -> j
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, -k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, -k, j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, i, k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -i, k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-k, -j}, []int{i, i, k, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -k, -j}, []int{i, i, k, j}, "queued", 0, 0}}
	case j == k && j != i && j != l && i != l:
		// x -> j, y -> i, z -> l
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{j, j, i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, -i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, -i, l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{j, j, -i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -l}, []int{j, j, i, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, -i, -l}, []int{j, j, i, l}, "queued", 0, 0}}
	case j == l && j != i && j != k && i != k:
		// x -> j, y -> i, z -> k
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{j, j, i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, -i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, -i, k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{j, j, i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-j, -j, i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{j, j, -i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -k}, []int{j, j, i, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-j, -j, -i, -k}, []int{j, j, i, k}, "queued", 0, 0}}
	case k == l && k != i && k != j && i != j:
		// x -> k, y -> i, z -> j
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{k, k, i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-k, -k, i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{k, k, -i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{-i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-k, -k, -i, j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{k, k, i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{2, HashName(), 0, 0, 0, []int{i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-k, -k, i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{k, k, -i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -j}, []int{k, k, i, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-k, -k, -i, -j}, []int{k, k, i, j}, "queued", 0, 0}}
	// 2 and 2
	case i == j && k == l && i != k:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, i}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{k, k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -i}, []int{i, i, k, k}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-k, -k}, []int{i, i, k, k}, "queued", 0, 0},
			Job{4, "E0", 0, 0, 0, []int{}, []int{i, i, k, k}, "queued", 0, 0}}
	case i == k && j == l && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{4, "E0", 0, 0, 0, []int{}, []int{i, i, j, j}, "queued", 0, 0}}
	case i == l && j == k && i != j:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -i, j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, i, -j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{i, i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{j, j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-i, -i}, []int{i, i, j, j}, "queued", 0, 0},
			Job{-2, HashName(), 0, 0, 0, []int{-j, -j}, []int{i, i, j, j}, "queued", 0, 0},
			Job{4, "E0", 0, 0, 0, []int{}, []int{i, i, j, j}, "queued", 0, 0}}
	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []Job{
			Job{1, HashName(), 0, 0, 0, []int{i, j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, -j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -j, k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, -j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -j, -k, l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, -j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, -j, k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{i, j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{i, -j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{-1, HashName(), 0, 0, 0, []int{-i, j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0},
			Job{1, HashName(), 0, 0, 0, []int{-i, -j, -k, -l}, []int{i, j, k, l}, "queued", 0, 0}}
	default:
		panic("No cases matched")
	}
}

func Derivative(dims ...int) []Job {
	switch len(dims) {
	case 2:
		return Make2D(dims[0], dims[1])
	case 3:
		return Make3D(dims[0], dims[1], dims[2])
	case 4:
		return Make4D(dims[0], dims[1], dims[2], dims[3])
	}
	return []Job{Job{}}
}
