package scheduler

import (
	"kjarmicki.github.com/cube/node"
	"kjarmicki.github.com/cube/task"
)

type Scheduler interface {
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

type RoudRobin struct {
	Name       string
	LastWorker int
}

func (rr *RoudRobin) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (rr *RoudRobin) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)
	var newWorker int
	if rr.LastWorker+1 < len(nodes) {
		rr.LastWorker += 1
	} else {
		rr.LastWorker = 0
	}
	newWorker = rr.LastWorker

	for idx, node := range nodes {
		if idx == newWorker {
			nodeScores[node.Name] = 1.0
		} else {
			nodeScores[node.Name] = 0.1
		}
	}
	return nodeScores
}

func (rr *RoudRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var lowestScore float64
	for idx, node := range candidates {
		if idx == 0 {
			bestNode = node
			lowestScore = scores[node.Name]
			continue
		}

		if scores[node.Name] < lowestScore {
			bestNode = node
			lowestScore = scores[node.Name]
		}
	}
	return bestNode
}
