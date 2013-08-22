package main

import (
	"runtime"
	"strings"
	"sync"
)

type Graph struct {
	Nodes map[string]*Node
	Mutex *sync.Mutex
}

type Node struct {
	Key        string
	Dependency *Dep
	Leaf       bool
	Nodes      map[string]*Node
}

func NewGraph() *Graph {
	return &Graph{Nodes: make(map[string]*Node), Mutex: &sync.Mutex{}}
}

func (graph *Graph) Insert(dependency *Dep) {
	graph.Mutex.Lock()
	keys := strings.Split(dependency.Import, "/")
	graph.Nodes[keys[0]] = deepInsert(graph.Nodes, keys, dependency)
	graph.Mutex.Unlock()
	runtime.Gosched()
}

func (graph *Graph) Search(importPath string) *Node {
	keys := strings.Split(importPath, "/")

	nodes := graph.Nodes
	for _, key := range keys {
		node := nodes[key]
		if node == nil {
			return nil
		}

		if node.Leaf {
			return node
		}

		nodes = node.Nodes
	}

	return nil
}

func deepInsert(nodes map[string]*Node, keys []string, dependency *Dep) *Node {
	node, found := nodes[keys[0]]
	if found == false {
		node = &Node{Key: keys[0], Nodes: make(map[string]*Node)}
	}

	newKeys := keys[1:]
	if len(newKeys) == 0 {
		node.Dependency = dependency
		node.Leaf = true
	} else {
		node.Nodes[newKeys[0]] = deepInsert(node.Nodes, newKeys, dependency)
	}

	return node
}
