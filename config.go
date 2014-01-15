package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	Checksum []byte
	// Path to the configuration file.
	Path string
	// Name of your repository "github.com/d2fn/gopack" for instance.
	Repository string
	// Dependencies tree
	DepsTree *toml.TomlTree
	// Whether the dependencies are vendorized or not
	Vendor bool
}

func NewConfig(dir string) *Config {
	return NewConfigFromFile(filepath.Join(dir, "gopack.config"))
}

func NewConfigFromFile(path string) *Config {
	config := &Config{Path: path}

	t, err := toml.LoadFile(config.Path)
	if err != nil {
		fail(err)
	}

	if deps := t.Get("deps"); deps != nil {
		config.DepsTree = deps.(*toml.TomlTree)
	}

	if repo := t.Get("repo"); repo != nil {
		config.Repository = repo.(string)
	}

	if vendor := t.Get("vendor"); vendor != nil {
		config.Vendor = vendor.(bool)
	}

	return config
}

func (c *Config) InitRepo(importGraph *Graph) {
	if c.Repository != "" {
		src := fmt.Sprintf("%s/%s/src", pwd, VendorDir)
		os.MkdirAll(src, 0755)

		dir := filepath.Dir(c.Repository)
		base := fmt.Sprintf("%s/%s", src, dir)
		os.MkdirAll(base, 0755)

		repo := fmt.Sprintf("%s/%s", src, c.Repository)
		err := os.Symlink(pwd, repo)
		if err != nil && !os.IsExist(err) {
			fail(err)
		}

		dependency := NewDependency(c.Repository)
		importGraph.Insert(dependency)
	}
}

func (c *Config) modifiedChecksum() bool {
	dat, err := ioutil.ReadFile(c.checksumPath())
	return (err != nil && os.IsNotExist(err)) || !bytes.Equal(dat, c.checksum())
}

func (c *Config) WriteChecksum() {
	os.MkdirAll(filepath.Join(pwd, GopackDir), 0755)
	err := ioutil.WriteFile(c.checksumPath(), c.checksum(), 0644)

	if err != nil {
		fail(err)
	}
}

func (c *Config) checksumPath() string {
	return filepath.Join(pwd, GopackChecksum)
}

func (c *Config) checksum() []byte {
	if c.Checksum == nil {
		dat, err := ioutil.ReadFile(c.Path)
		if err != nil {
			fail(err)
		}

		h := md5.New()
		h.Write(dat)
		c.Checksum = h.Sum(nil)
	}
	return []byte(hex.EncodeToString(c.Checksum))
}

func (c *Config) LoadDependencyModel(importGraph *Graph) (deps *Dependencies, err error) {
	depsTree := c.DepsTree

	if depsTree == nil {
		return
	}

	modifiedChecksum := c.modifiedChecksum()
	deps = NewDependencies(importGraph, len(depsTree.Keys()))

	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		d := NewDependency(depTree.Get("import").(string))

		d.setScm(depTree)
		d.setSource(depTree)

		d.setCheckout(depTree, "branch", BranchFlag)
		d.setCheckout(depTree, "commit", CommitFlag)
		d.setCheckout(depTree, "tag", TagFlag)

		if err := d.Validate(); err != nil {
			return nil, err
		}

		d.Fetch(modifiedChecksum)

		deps.Save(i, k, d)
	}

	return deps, nil
}

func (c *Config) WriteVendor() error {
	content, err := ioutil.ReadFile(c.Path)
	if err != nil {
		return err
	}

	err = os.Rename(c.Path, filepath.Join(pwd, VendorDir, GopackLock))
	if err != nil {
		return err
	}

	newContent := fmt.Sprintf("# Dependencies vendored.\n# Do not remove this option.\nvendor = true\n%s", string(content))
	return ioutil.WriteFile(c.Path, []byte(newContent), 0755)
}
