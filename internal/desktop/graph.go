package desktop

import "supergit/internal/gitrepo"

type graphEdge struct{ Row, Lane, ViaLane int }
type graphRow struct {
	Lane     int
	Edges    []graphEdge
	Boundary bool
}

func layoutGraph(commits []gitrepo.Commit) []graphRow {
	rows := make([]graphRow, len(commits))
	positions := make(map[string]int, len(commits))
	for i, c := range commits {
		positions[c.SHA] = i
	}
	var lanes []string
	find := func(sha string) int {
		for i, s := range lanes {
			if s == sha {
				return i
			}
		}
		return -1
	}
	allocate := func(sha string) int {
		for i, s := range lanes {
			if s == "" {
				lanes[i] = sha
				return i
			}
		}
		lanes = append(lanes, sha)
		return len(lanes) - 1
	}
	for i, c := range commits {
		lane := find(c.SHA)
		if lane < 0 {
			lane = allocate(c.SHA)
		}
		rows[i].Lane = lane
		for j, sha := range lanes {
			if sha == c.SHA {
				lanes[j] = ""
			}
		}
		for parentIndex, parent := range c.Parents {
			row, ok := positions[parent]
			if !ok || row <= i {
				rows[i].Boundary = true
				continue
			}
			via := find(parent)
			if parentIndex == 0 {
				via = lane
				lanes[via] = parent
			} else if via < 0 {
				via = allocate(parent)
			}
			rows[i].Edges = append(rows[i].Edges, graphEdge{Row: row, ViaLane: via})
		}
	}
	for i := range rows {
		for j := range rows[i].Edges {
			rows[i].Edges[j].Lane = rows[rows[i].Edges[j].Row].Lane
		}
	}
	return rows
}
