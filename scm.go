package main

// LOL so we're gonna try and avoid THIS situation http://golang.org/src/cmd/go/vcs.go#L331

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	GitTag    = "git"
	HgTag     = "hg"
	SvnTag    = "svn"
	BzrTag    = "bzr"
	HiddenGit = ".git"
	HiddenHg  = ".hg"
	HiddenSvn = ".svn"
	HiddenBzr = ".bzr"
)

type Scm interface {
	Init(d *Dep) error
	Checkout(d *Dep) error
	Fetch(path string) error
	DownloadCommand(source, path string) *exec.Cmd
	WriteVendorIgnores() error
}

var (
	Scms = map[string]Scm{
		GitTag: Git{},
		HgTag:  Hg{},
		SvnTag: Svn{},
		BzrTag: Bzr{}}

	HiddenDirs = map[string]string{
		GitTag: HiddenGit,
		HgTag:  HiddenHg,
		SvnTag: HiddenSvn,
		BzrTag: HiddenBzr}
)

func dependencyPath(importPath string) string {
	return path.Join(pwd, VendorDir, "src", importPath)
}

func scmStageDir(depPath, scmDir string) string {
	return path.Join(depPath, scmDir)
}

func downloadDependency(d *Dep, depPath, scmType string, scm Scm) (err error) {
	stage, err := os.Stat(scmStageDir(depPath, scmType))

	if stage != nil && stage.IsDir() {
		err = scm.Fetch(depPath)
	} else if err != nil && !os.IsNotExist(err) {
		err = fmt.Errorf("Error while examining dependency path for %s: %s", d.Import, err)
	} else {
		fmtcolor(Gray, "downloading %s\n", d.Source)

		cmd := scm.DownloadCommand(d.Source, depPath)

		if err = cmd.Run(); err != nil {
			return fmt.Errorf("Error downloading dependency: %s", err)
		}
	}

	return
}

func initScm(d *Dep, scmType string, scm Scm) error {
	path := dependencyPath(d.Import)

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("Error creating import dir %s", err)
	} else {
		return downloadDependency(d, path, scmType, scm)
	}
}

func runInPath(path string, fn func() error) error {
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	defer os.Chdir(pwd)

	return fn()
}

type Git struct{}

func (g Git) Init(d *Dep) error {
	return initScm(d, HiddenGit, g)
}

func (g Git) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("git", "clone", source, path)
}

func (g Git) Checkout(d *Dep) error {
	cmd := exec.Command("git", "checkout", d.CheckoutSpec)
	return cmd.Run()
}

func (g Git) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("git", "fetch").Run()
	})
}

func (g Git) WriteVendorIgnores() error {
	gitignore := path.Join(pwd, VendorDir, ".gitignore")
	os.MkdirAll(filepath.Dir(gitignore), 0755)

	return ioutil.WriteFile(gitignore, []byte("-/bin\n-/pkg\n"), 0755)
}

type Hg struct{}

func (h Hg) Init(d *Dep) error {
	return initScm(d, HiddenHg, h)
}

func (h Hg) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("hg", "clone", source, path)
}

func (h Hg) Checkout(d *Dep) error {
	var cmd *exec.Cmd

	if d.CheckoutFlag == CommitFlag {
		cmd = exec.Command("hg", "update", "-c", d.CheckoutSpec)
	} else {
		cmd = exec.Command("hg", "checkout", d.CheckoutSpec)
	}

	return cmd.Run()
}

func (h Hg) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("hg", "pull").Run()
	})
}

func (h Hg) WriteVendorIgnores() (err error) {
	file, err := os.OpenFile(path.Join(pwd, ".hgignore"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("\nsyntax: glob\n%s\n%s\n",
		path.Join(VendorDir, "bin"), path.Join(VendorDir, "pkg")))
	return
}

type Svn struct{}

func (s Svn) Init(d *Dep) error {
	return initScm(d, HiddenSvn, s)
}

func (s Svn) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("svn", "checkout", source, path)
}

func (s Svn) Checkout(d *Dep) error {
	var cmd *exec.Cmd

	switch d.CheckoutFlag {
	case CommitFlag:
		cmd = exec.Command("svn", "up", "-r", d.CheckoutSpec)
	case BranchFlag:
		cmd = exec.Command("svn", "switch", "^/branches/"+d.CheckoutSpec)
	case TagFlag:
		cmd = exec.Command("svn", "switch", "^/tags/"+d.CheckoutSpec)
	}

	return cmd.Run()
}

func (s Svn) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("svn", "update").Run()
	})
}

func (s Svn) WriteVendorIgnores() (err error) {
	err = exec.Command("svn", "propset", "svn:ignore", path.Join(VendorDir, "bin"), ".").Run()
	if err != nil {
		return
	}
	err = exec.Command("svn", "propset", "svn:ignore", path.Join(VendorDir, "pkg"), ".").Run()
	return
}

type Bzr struct{}

func (b Bzr) Init(d *Dep) error {
	return initScm(d, HiddenBzr, b)
}

func (b Bzr) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("bzr", "branch", source, path)
}

func (b Bzr) Checkout(d *Dep) error {
	var cmd *exec.Cmd

	switch d.CheckoutFlag {
	case CommitFlag:
		cmd = exec.Command("bzr", "update", "-r", d.CheckoutSpec)
	case BranchFlag:
		cmd = exec.Command("bzr", "update", "-r", "branch:"+d.CheckoutSpec)
	case TagFlag:
		cmd = exec.Command("bzr", "update", "-r", "tag:"+d.CheckoutSpec)
	}

	return cmd.Run()
}

func (b Bzr) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("bzr", "pull").Run()
	})
}

func (b Bzr) WriteVendorIgnores() (err error) {
	err = exec.Command("bzr", "ignore", path.Join(VendorDir, "bin")).Run()
	if err != nil {
		return
	}
	err = exec.Command("bzr", "ignore", path.Join(VendorDir, "pkg")).Run()
	return
}

// The Go scm embeds another scm and only implements Init so that
// deps that don't specify a scm keep working like they did before
type Go struct {
	Scm
}

func (g Go) Init(d *Dep) error {
	return g.DownloadCommand(d.Import, "").Run()
}

func (g Go) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("go", "get", "-d", "-u", source)
}

func NewScm(d *Dep) (Scm, error) {
	switch d.Scm {
	case GitTag:
		return Scms[GitTag], nil
	case HgTag:
		return Scms[HgTag], nil
	case SvnTag:
		return Scms[SvnTag], nil
	}

	scm := scmInSource(d)

	if d.Scm == "go" {
		return Go{scm}, nil
	} else if scm != nil {
		return scm, nil
	}

	return nil, fmt.Errorf("unknown scm for %s", d.Import)
}

// Traverse the source tree backwards until
// it finds the right directory
// or it arrives to the base of the import.
func scmInSource(d *Dep) Scm {
	parts := strings.Split(d.Import, "/")
	initPath := d.Src()

	for _, _ = range parts {
		if scm := scmInPath(initPath); scm != nil {
			return scm
		}
		initPath = path.Join(initPath, "..")
	}

	return nil
}

func scmInPath(initPath string) Scm {
	for key, scm := range Scms {
		if isDir(path.Join(initPath, HiddenDirs[key])) {
			return scm
		}
	}
	return nil
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
