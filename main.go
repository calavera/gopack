package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	GopackVersion      = "0.20.dev"
	GopackDir          = ".gopack"
	GopackChecksum     = ".gopack/checksum"
	GopackTestProjects = ".gopack/test-projects"
	VendorDir          = ".gopack/vendor"
	VendorSrcDir       = ".gopack/vendor/src"
	GopackLock         = "gopack.lock"
)

const (
	Blue     = uint8(94)
	Green    = uint8(92)
	Red      = uint8(31)
	Gray     = uint8(90)
	EndColor = "\033[0m"
)

var (
	pwd        string
	showColors = true
)

func main() {
	if os.Getenv("GOPACK_SKIP_COLORS") == "1" {
		showColors = false
	}

	// localize GOPATH
	setupEnv()

	p, err := AnalyzeSourceTree(".")
	if err != nil {
		fail(err)
	}

	config, deps := loadDependencies(".", p)

	if deps == nil {
		fail("Error loading dependency info")
	}

	switch os.Args[1] {
	case "dependencytree":
		deps.PrintDependencyTree()
	case "stats":
		p.PrintSummary()
	case "installdeps":
		deps.Install(config.Repository)
	case "vendor":
		vendorDependencies(config, deps)
		fmtcolor(Gray, "Vendor dependencies ready\n")
	default:
		runCommand()
	}
}

func loadDependencies(root string, p *ProjectStats) (*Config, *Dependencies) {
	config, dependencies := loadConfiguration(root)
	if dependencies != nil && !config.Vendor {
		announceGopack()
		failWith(dependencies.Validate(p))
		// prepare dependencies
		loadTransitiveDependencies(dependencies)
		config.WriteChecksum()
	}
	return config, dependencies
}

func loadConfiguration(dir string) (*Config, *Dependencies) {
	importGraph := NewGraph()
	config := NewConfig(dir)
	config.InitRepo(importGraph)

	dependencies, err := config.LoadDependencyModel(importGraph)
	if err != nil {
		failf(err.Error())
	}
	return config, dependencies
}

func runCommand() {
	first := os.Args[1]
	if first == "version" {
		fmt.Printf("gopack version %s\n", GopackVersion)
		os.Exit(0)
	}

	run(os.Args[1:]...)
}

func run(args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fail(err)
	}
}

func loadTransitiveDependencies(dependencies *Dependencies) {
	fetchTransitiveDependencies(dependencies, false)
}

func fetchTransitiveDependencies(dependencies *Dependencies, clean bool) {
	dependencies.VisitDeps(
		func(dep *Dep) {
			fmtcolor(Gray, "updating %s\n", dep.Import)
			if clean {
				depStats, err := AnalyzeSourceTree(dep.Src())
				if err != nil {
					fail(err)
				}
				dep.CleanSrc()

				for path, s := range depStats.ImportStatsByPath {
					if s.Remote && !s.Test && !strings.HasPrefix(path, dep.Import) {
						//FIXME: This is a really naive implementation
						srcDir := filepath.Join(pwd, VendorSrcDir, path)
						os.RemoveAll(srcDir)
					}
				}
			}
			dep.Get()

			if dep.CheckoutType() != "" {
				fmtcolor(Gray, "pointing %s at %s %s\n", dep.Import, dep.CheckoutType(), dep.CheckoutSpec)
				dep.switchToBranchOrTag()
			}

			if dep.fetch {
				transitive, err := dep.LoadTransitiveDeps(dependencies.ImportGraph)
				if err != nil {
					failf(err.Error())
				}
				if transitive != nil {
					fetchTransitiveDependencies(transitive, clean)
				}
			}
		})
}

// Set the working directory.
// It's the current directory by default.
// It can be overriden setting the environment variable GOPACK_APP_CONFIG.
func setPwd() {
	var dir string
	var err error

	dir = os.Getenv("GOPACK_APP_CONFIG")
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			fail(err)
		}
	}

	pwd = dir
}

// set GOPATH to the local vendor dir
func setupEnv() {
	setPwd()
	vendor := fmt.Sprintf("%s/%s", pwd, VendorDir)
	err := os.Setenv("GOPATH", vendor)
	if err != nil {
		fail(err)
	}
}

func fmtcolor(c uint8, s string, args ...interface{}) {
	if showColors {
		fmt.Printf("\033[%dm", c)
	}

	if len(args) > 0 {
		fmt.Printf(s, args...)
	} else {
		fmt.Printf(s)
	}

	if showColors {
		fmt.Printf(EndColor)
	}
}

func vendorDependencies(config *Config, deps *Dependencies) {
	pristine := true
	var err error

	if config.Vendor {
		err = updateVendoredDependencies(config, deps)
		if err != nil {
			failf(err.Error())
		}
		pristine = false
	}

	mainScm := scmInPath(pwd)
	if mainScm == nil {
		failf(fmt.Sprintf("Unknown scm at %s\n", pwd))
	}

	err = cleanScms()
	if err != nil {
		failf(err.Error())
	}

	if pristine {
		err = mainScm.WriteVendorIgnores()
		if err != nil {
			failf(err.Error())
		}
	}

	err = config.WriteVendor()
	if err != nil {
		failf(err.Error())
	}
}

func updateVendoredDependencies(config *Config, deps *Dependencies) (err error) {
	lockConfig := NewConfigFromFile(filepath.Join(pwd, VendorDir, GopackLock))
	lockGraph := NewGraph()
	_, err = lockConfig.LoadDependencyModel(lockGraph)
	if err != nil {
		return
	}

	diffMap := make(map[string]*Dep)
	for _, dep := range deps.DepList {
		node := lockGraph.Search(dep.Import)
		if node == nil || dep.Diff(node.Dependency) {
			dep.Fetch(true)
			diffMap[dep.Import] = dep
		}
	}

	if len(diffMap) == 0 {
		return
	}

	diffDeps := NewDependencies(NewGraph(), len(diffMap))
	diffIndex := 0
	for key, dep := range diffMap {
		diffDeps.Save(diffIndex, key, dep)
		diffIndex++
	}

	if len(diffDeps.DepList) > 0 {
		announceGopack()
		fetchTransitiveDependencies(diffDeps, true)
	}

	return
}

func cleanScms() error {
	srcDir := filepath.Join(pwd, VendorSrcDir)

	return filepath.Walk(srcDir, func(path string, fi os.FileInfo, err error) error {
		name := fi.Name()

		if name == "bin" || name[0] == '.' || name[0] == '_' {
			if rErr := os.RemoveAll(path); rErr != nil {
				log.Printf("Unable to clean dependency path: %s\n", path)
				return rErr
			}
		}
		return nil
	})
}

func logcolor(c uint8, s string, args ...interface{}) {
	log.Printf("\033[%dm", c)
	if len(args) > 0 {
		log.Printf(s, args...)
	} else {
		log.Printf(s)
	}
	log.Printf(EndColor)
}

func failf(s string, args ...interface{}) {
	fmtcolor(Red, s, args...)
	os.Exit(1)
}

func fail(a ...interface{}) {
	fmt.Printf("\033[%dm", Red)
	fmt.Print(a)
	fmt.Printf(EndColor)
	os.Exit(1)
}

func failWith(errors []*ProjectError) {
	if len(errors) > 0 {
		fmt.Printf("\033[%dm", Red)
		for _, e := range errors {
			fmt.Printf(e.String())
		}
		fmt.Printf(EndColor)
		fmt.Println()
		os.Exit(len(errors))
	}
}

func announceGopack() {
	fmtcolor(104, "/// g o p a c k ///")
	fmt.Println()
}
