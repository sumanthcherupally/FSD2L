package snapshot

import (
	dt "GO-DAG/DataTypes"
	"encoding/hex"
	// "math/rand"
)

// PruneDag prunes the Old Transactions to reduce memory overhead
func PruneDag(dag *dt.DAG, Ratings map[string]int, Threshold int) {

	for k, v := range dag.Graph {
		if Ratings[k] > Threshold {
			for _, neighbour := range v.Neighbours {
				clearTips(dag.Graph[neighbour], k)
			}
			delete(dag.Graph, k)
		}
	}

	return
}

func clearTips(v dt.Vertex, tip string) {
	var t [32]byte
	if hex.EncodeToString(v.Tx.LeftTip[:]) == tip {
		v.Tx.LeftTip = t
	}
	if hex.EncodeToString(v.Tx.RightTip[:]) == tip {
		v.Tx.RightTip = t
	}
	return
}

func getMaxSet(Ratings map[string]int, neighbours []string) string {
	step := neighbours[0]
	max := Ratings[neighbours[0]]
	for _, v := range neighbours {
		if Ratings[v] > max {
			max = Ratings[v]
			step = v
		}
	}
	return step
}

func getNewMilestone(Ratings map[string]int, dag *dt.DAG) string {
	prev := dag.Genisis
	path := dag.Genisis
	for {
		neighbours := dag.Graph[path].Neighbours
		path = getMaxSet(Ratings, neighbours)
		if Ratings[path] < 1000 {
			break
		}
		prev = path
	}
	return prev
}
