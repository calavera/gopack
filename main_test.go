package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestSetPwdDefault(t *testing.T) {
	os.Setenv("GOPACK_APP_CONFIG", "")
	setPwd()
	dir, _ := os.Getwd()
	if pwd != dir {
		t.Errorf("Expected pwd to be %s but it was %s.\n", dir, pwd)
	}
}

func TestSetPwdAppConfig(t *testing.T) {
	dir, _ := ioutil.TempDir("", "gopack-test-")
	os.Setenv("GOPACK_APP_CONFIG", dir)
	setPwd()
	if pwd != dir {
		t.Errorf("Expected pwd to be %s but it was %s.\n", dir, pwd)
	}
}

func TestCleanScmFiles(t *testing.T) {
	setupTestPwd()

	dep := createScmDep(HiddenGit, "github.com/d2fn/gopack", ".git/objects")

	gitDir := path.Join(dep.Src(), ".git")
	hiddenFile := path.Join(dep.Src(), ".gitignore")
	underscoreFile := path.Join(dep.Src(), "__file__")

	ioutil.WriteFile(hiddenFile, []byte("foo\nbar"), 0755)
	ioutil.WriteFile(underscoreFile, []byte("foo\nbar"), 0755)

	cleanScms()

	_, err := os.Stat(gitDir)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("Expected %s to not exist: %v", gitDir, err)
	}

	_, err = os.Stat(hiddenFile)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("Expected %s to not exist: %v", hiddenFile, err)
	}

	_, err = os.Stat(underscoreFile)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("Expected %s to not exist: %v", underscoreFile, err)
	}
}
