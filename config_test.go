package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func createFixtureConfig(dir string, config string) {
	err := ioutil.WriteFile(path.Join(dir, "gopack.config"), []byte(config), 0644)
	check(err)
}

func setupTestConfig(fixture string) *Config {
	setupTestPwd()
	setupEnv()

	createFixtureConfig(pwd, fixture)
	return NewConfig(pwd)
}

func TestNewConfig(t *testing.T) {
	config := setupTestConfig(`
repo = "github.com/d2fn/gopack"

[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  branch = "master"
`)

	if config.Repository == "" {
		t.Error("Expected repository to not be empty.")
	}

	if config.DepsTree == nil {
		t.Error("Expected dependency tree to not be empty.")
	}
}

func TestInitRepoWithoutRepo(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  branch = "master"
`)

	graph := NewGraph()
	config.InitRepo(graph)

	src := path.Join(pwd, VendorDir, "src")
	_, err := os.Stat(src)

	if !os.IsNotExist(err) {
		t.Errorf("Expected vendor to not exist in %s\n", pwd)
	}
}

func TestInitRepo(t *testing.T) {
	config := setupTestConfig(`repo = "github.com/d2fn/gopack"`)

	graph := NewGraph()
	config.InitRepo(graph)

	dep := path.Join(pwd, VendorDir, "src", "github.com", "d2fn", "gopack")
	stat, err := os.Stat(dep)

	if os.IsNotExist(err) || (stat.Mode()&os.ModeSymlink != 0) {
		t.Errorf("Expected repository %s to be linked in vendor %s\n", config.Repository, pwd)
	}

	if graph.Search(config.Repository) == nil {
		t.Errorf("Expected repository %s to be in the dependencies graph\n", config.Repository)
	}
}

func TestWriteChecksum(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  branch = "master"
`)

	config.WriteChecksum()

	path := path.Join(pwd, GopackChecksum)
	_, err := ioutil.ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		t.Errorf("Expected checksum file %s to exist", path)
	}
}

func TestFetchDependenciesWithoutChecksum(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  branch = "master"
`)

	if cfg, _ := config.LoadDependencyModel(NewGraph()); !cfg.AllDepsNeedFetching() {
		t.Errorf("Expected to load all the dependencies when there is no checksum")
	}
}

func TestFetchDependenciesWithoutChanges(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
`)
	config.WriteChecksum()

	deps, _ := config.LoadDependencyModel(NewGraph())
	if deps.AnyDepsNeedFetching() {
		t.Errorf("Expected to not load any dependency with commit flag")
	}
}

func TestFetchDependenciesWithBranch(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  branch = "master"
`)
	config.WriteChecksum()

	deps, _ := config.LoadDependencyModel(NewGraph())
	if len(deps.DepList) != 1 {
		t.Errorf("Expected to load any dependency with branch flag")
	}
}

func TestFetchDependenciesWithChanges(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
`)

	config.WriteChecksum()
	config.Checksum = nil

	fixture := `
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
`
	createFixtureConfig(pwd, fixture)

	deps, _ := config.LoadDependencyModel(NewGraph())
	if len(deps.DepList) != 1 {
		t.Errorf("Expected to load only the new dependencies")
	}
}

func TestFetchWithCommitSpecs(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
`)
	config.WriteChecksum()

	deps, _ := config.LoadDependencyModel(NewGraph())
	if deps.DepList[0].fetch {
		t.Errorf("Expected to not fetch the commit dependencies")
	}
	if !deps.DepList[1].fetch {
		t.Errorf("Expected to fetch the branch dependencies")
	}
}

func TestFetchWithTagSpecs(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  tag = "v1.0.0"
[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
`)
	config.WriteChecksum()

	deps, _ := config.LoadDependencyModel(NewGraph())
	if deps.DepList[0].fetch {
		t.Errorf("Expected to not fetch the tag dependencies")
	}
	if !deps.DepList[1].fetch {
		t.Errorf("Expected to fetch the branch dependencies")
	}
}

func TestFetchWithMixedSpecsIgnoringOrder(t *testing.T) {
	config := setupTestConfig(`
[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
`)
	config.WriteChecksum()

	deps, _ := config.LoadDependencyModel(NewGraph())
	if !deps.DepList[0].fetch {
		t.Errorf("Expected to not fetch the commit dependencies")
	}
	if deps.DepList[1].fetch {
		t.Errorf("Expected to fetch the branch dependencies")
	}
}

func TestWriteVendor(t *testing.T) {
	config := setupTestConfig(`
[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
`)

	err := config.WriteVendor()
	if err != nil {
		t.Fatal(err)
	}

	expected := `# Dependencies vendored.
# Do not remove this option.
vendor = true

[deps.foo]
  import = "github.com/calavera/foo"
  branch = "master"
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
  commit = "182cae2ee3926a960223d8db4998aa9d57c89788"
`

	bytes, err := ioutil.ReadFile(config.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bytes)

	if content != expected {
		t.Fatalf("Expected config:\n%s\n\nbut it was:\n%s", expected, content)
	}
}
