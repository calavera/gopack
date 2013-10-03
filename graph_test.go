package main

import (
	"strings"
	"testing"
)

func TestDeepGraph(t *testing.T) {
	graph := NewGraph()
	dep1 := &Dep{Import: "github.com/d2fn/gopack"}
	dep2 := &Dep{Import: "code.google.com/p/go.net"}

	graph.Insert(dep1)
	graph.Insert(dep2)

	testTree(dep1, graph, t)
	testTree(dep2, graph, t)
}

func testTree(dep *Dep, graph *Graph, t *testing.T) {
	nodes := graph.Nodes
	keys := strings.Split(dep.Import, "/")

	for idx, key := range keys {
		node := nodes[key]
		if node == nil {
			t.Error("Expected node to not be nil")
		}

		if idx < len(keys)-1 {
			if node.Leaf == true {
				t.Error("Expected leaf to not be a leaf")
			}

			if node.Dependency != nil {
				t.Errorf("Expected node to not store the dependency")
			}

			nodes = node.Nodes
		} else {
			if node.Leaf == false {
				t.Error("Expected node to be a leaf")
			}

			if node.Dependency != dep {
				t.Errorf("Expected node to store the dependency")
			}
		}
	}
}

func TestSearchFailsWithNoNodes(t *testing.T) {
	graph := NewGraph()
	node := graph.Search("github.com/d2fn/gopack")

	if node != nil {
		t.Error("Expected search to fail when there are no nodes")
	}
}

func TestSearchFailsWithDifferentNodes(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/dotcloud/docker")
	if node != nil {
		t.Error("Expected search to fail when the dependency doesn't exist")
	}
}

func TestSearchWorksWithBareNames(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/d2fn/gopack")
	if node == nil {
		t.Error("Expected search to succeed importing bare repos")
	}
}

func TestSearchWorksWithExtendedNames(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/d2fn/gopack/graph")
	if node.Dependency != dep {
		t.Error("Expected search to succeed importing extended repos")
	}
}

func TestValidDepWhenDoesntExist(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}

	if _, ok := graph.Valid(dep); !ok {
		t.Error("Expected dependency to be valid when the graph doesn't include it")
	}
}

func TestInvalidWithDifferentFlags(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: BranchFlag}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be invalid when the checkout flag is different")
	}

	dep3 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: CommitFlag}
	if _, ok := graph.Valid(dep3); ok {
		t.Error("Expected dependency to be invalid when the checkout flag is different")
	}
}

func TestInvalidWithDifferentSpecs(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: CommitFlag, CheckoutSpec: "asdf"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: CommitFlag, CheckoutSpec: "qwert"}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be invalid when the checkout spec is different")
	}
}

func TestValidWithSameRoot(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack/foo", CheckoutFlag: CommitFlag, CheckoutSpec: "asdf"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack/bar", CheckoutFlag: CommitFlag, CheckoutSpec: "asdf"}
	if _, ok := graph.Valid(dep2); !ok {
		t.Error("Expected dependency to be valid when the dependencies have the same root")
	}
}

func TestInvalidWithDifferentRoot(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack/foo", CheckoutFlag: CommitFlag, CheckoutSpec: "asdf"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack/bar", CheckoutFlag: CommitFlag, CheckoutSpec: "qwert"}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be valid when the dependencies have the same root")
	}
}

func TestValidWithSameTagVersioning(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.0"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.0"}
	if _, ok := graph.Valid(dep2); !ok {
		t.Error("Expected dependency to be valid when the dependency tag spec is the same")
	}
}

func TestValidWithPesimisticTagVersioning(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.8"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.0"}
	if _, ok := graph.Valid(dep2); !ok {
		t.Error("Expected dependency to be valid when the dependency tag spec is pesimitically greater")
	}
}

func TestInvalidWithInvalidPesimisticTagVersioning(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.0"}
	graph.Insert(dep)

	dep2 := &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.2.8"}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be invalid when the dependency tag spec is pesimitically lower")
	}

	dep2 = &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.1.8"}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be invalid when the dependency tag spec is pesimitically lower")
	}

	dep2 = &Dep{Import: "github.com/d2fn/gopack", CheckoutFlag: TagFlag, CheckoutSpec: "v1.3.8"}
	if _, ok := graph.Valid(dep2); ok {
		t.Error("Expected dependency to be invalid when the dependency tag spec is pesimitically lower")
	}
}
