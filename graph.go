package main

import (
	"github.com/jm/go-semver"
	"regexp"
	"strings"
)

type Graph struct {
	Nodes map[string]*Node
}

type Node struct {
	Key        string
	Dependency *Dep
	Leaf       bool
	Nodes      map[string]*Node
}

// Borrowed from https://code.google.com/p/go/source/browse/src/cmd/go/vcs.go
var vcsRegexps = []*regexp.Regexp{
	// Google Code - new syntax
	regexp.MustCompile(`^(?P<root>code\.google\.com/p/(?P<project>[a-z0-9\-]+)(\.(?P<subrepo>[a-z0-9\-]+))?)(/[A-Za-z0-9_.\-]+)*$`),
	// Github
	regexp.MustCompile(`^(?P<root>github\.com/[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)(/[A-Za-z0-9_.\-]+)*$`),
	// Bitbucket
	regexp.MustCompile(`^(?P<root>bitbucket\.org/(?P<bitname>[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`),
	// Launchpad
	regexp.MustCompile(`^(?P<root>launchpad\.net/((?P<project>[A-Za-z0-9_.\-]+)(?P<series>/[A-Za-z0-9_.\-]+)?|~[A-Za-z0-9_.\-]+/(\+junk|[A-Za-z0-9_.\-]+)/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`),
	// General syntax for any server
	regexp.MustCompile(`^(?P<root>(?P<repo>([a-z0-9.\-]+\.)+[a-z0-9.\-]+(:[0-9]+)?/[A-Za-z0-9_.\-/]*?)\.(?P<vcs>bzr|git|hg|svn))(/[A-Za-z0-9_.\-]+)*$`),
}

func NewGraph() *Graph {
	return &Graph{Nodes: make(map[string]*Node)}
}

func (graph *Graph) Insert(dependency *Dep) {
	keys := graph.importParts(dependency.Import)
	graph.Nodes[keys[0]] = deepInsert(graph.Nodes, keys, dependency)
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

func (graph *Graph) PreOrderVisit(fn func(n *Node, depth int)) {
	for _, node := range graph.Nodes {
		node.PreOrderVisit(fn, 0)
	}
}

func (parent *Node) PreOrderVisit(fn func(n *Node, depth int), depth int) {
	for _, node := range parent.Nodes {
		fn(node, depth)
		if !node.Leaf {
			node.PreOrderVisit(fn, depth+1)
		}
	}
}

func (graph *Graph) Valid(dependency *Dep) (node *Node, valid bool) {
	node = graph.Search(dependency.Import)
	valid = node == nil || graph.validSpec(node.Dependency, dependency)

	return
}

func (graph *Graph) validSpec(d1, d2 *Dep) bool {
	if d1.CheckoutFlag == TagFlag && d2.CheckoutFlag == TagFlag {
		v1 := strings.Split(d1.CheckoutSpec, "v")
		v2 := strings.Split(d2.CheckoutSpec, "v")
		s1 := semver.FromString(v1[len(v1)-1])
		s2 := semver.FromString(v2[len(v2)-1])

		return s1.PessimisticGreaterThan(s2)
	} else {
		return d1.CheckoutFlag == d2.CheckoutFlag &&
			d1.CheckoutSpec == d2.CheckoutSpec
	}
}

func (graph *Graph) importParts(importPath string) []string {
	for _, re := range vcsRegexps {
		match := re.FindStringSubmatch(importPath)
		if match != nil {
			return strings.Split(match[1], "/")
		}
	}
	return strings.Split(importPath, "/")
}
