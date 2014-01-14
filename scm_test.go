package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestGitWriteIgnores(t *testing.T) {
	setupTestPwd()

	git := Git{}
	err := git.WriteVendorIgnores()
	if err != nil {
		t.Fatal(err)
	}

	ignore := path.Join(pwd, VendorDir, ".gitignore")
	_, err = os.Stat(ignore)
	if err != nil {
		t.Fatalf("Expected git ignore file to exist: %s", ignore)
	}

	content, err := ioutil.ReadFile(ignore)
	if err != nil {
		t.Fatalf("Expected to be able to read git ignore: %s", ignore)
	}

	expected := "-/bin\n-/pkg\n"
	if string(content) != expected {
		t.Fatalf("Expected to ignore pkg and bin:\n%s\nbut it was:\n%s", expected, content)
	}
}

func TestHgWriteIgnores(t *testing.T) {
	setupTestPwd()

	hg := Hg{}
	err := hg.WriteVendorIgnores()
	if err != nil {
		t.Fatal(err)
	}

	ignore := path.Join(pwd, ".hgignore")
	_, err = os.Stat(ignore)
	if err != nil {
		t.Fatalf("Expected hg ignore file to exist: %s", ignore)
	}

	fmt.Println(ignore)
	content, err := ioutil.ReadFile(ignore)
	if err != nil {
		t.Fatalf("Expected to be able to read hg ignore: %s", ignore)
	}

	expected := fmt.Sprintf("\nsyntax: glob\n%s\n%s\n",
		path.Join(VendorDir, "bin"), path.Join(VendorDir, "pkg"))

	if string(content) != expected {
		t.Fatalf("Expected to ignore pkg and bin:\n%s\nbut it was:\n%s", expected, content)
	}
}

func TestHgAppendsIgnores(t *testing.T) {
	setupTestPwd()
	ioutil.WriteFile(path.Join(pwd, ".hgignore"), []byte("*.a"), 0755)

	hg := Hg{}
	err := hg.WriteVendorIgnores()
	if err != nil {
		t.Fatal(err)
	}

	ignore := path.Join(pwd, ".hgignore")
	_, err = os.Stat(ignore)
	if err != nil {
		t.Fatalf("Expected hg ignore file to exist: %s", ignore)
	}

	fmt.Println(ignore)
	content, err := ioutil.ReadFile(ignore)
	if err != nil {
		t.Fatalf("Expected to be able to read hg ignore: %s", ignore)
	}

	expected := fmt.Sprintf("*.a\nsyntax: glob\n%s\n%s\n",
		path.Join(VendorDir, "bin"), path.Join(VendorDir, "pkg"))

	if string(content) != expected {
		t.Fatalf("Expected to ignore pkg and bin:\n%s\nbut it was:\n%s", expected, content)
	}
}
