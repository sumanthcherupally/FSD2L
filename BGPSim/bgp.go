package main

/*
func simBGP(NumNodes int, file *os.File) {
	var prefixes []string
	var prefixAllocate map[string]int
	var prefixPaths map[string]map[int][]int

	prefixAllocate = make(map[string]int)
	prefixPaths = make(map[string]map[int][]int)

	subPrefix := "192.168.0"

	for i := 1; i < 255; i++ {
		p := strconv.Itoa(i) + "/24"
		prefixes = append(prefixes, subPrefix+p)
	}

	// write the allocation for each prefix
	perm := rand.Perm(254)
	for i := 0; i < 254; i++ {
		prefixAllocate[prefixes[perm[i]]] = i % (NumNodes)
	}

	for i := 0; i < len(prefixes); i++ {
		prefix := prefixes[i]
		prefixPaths[prefix] = make(map[int][]int)
		currentAS := prefixAllocate[prefix]

		randomAS := rand.Perm(NumNodes)
		count := 0
		var adjacentNodes []int

		for j := 0; j < 5; j++ {

			randLenAdj := rand.Intn(3) + 1

			for k := 0; k < randLenAdj; k++ {
				if randomAS[count] == currentAS {

				}
			}

		}

	}

}
*/
