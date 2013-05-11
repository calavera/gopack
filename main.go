package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	VendorDir = "vendor"
)

var (
	pwd string
)

func main() {
	setupEnv()
	log.Println(pwd)
	d := LoadDependencyModel()
	// prepare dependencies
	d.VisitDeps(
		func(d *Dep) {
			log.Printf("updating %s\n", d.Import)
			d.goGetUpdate()
			log.Printf("pointing %s at %s %s\n", d.Import, d.CheckoutType(), d.CheckoutSpec)
			d.switchToBranchOrTag()
		})
	// run the specified command
	cmd := exec.Command("go", os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}

// set GOPATH to the local vendor dir
func setupEnv() {
	dir, err := os.Getwd()
	pwd = dir
	log.Println(pwd)
	if err != nil {
		log.Fatal(err)
	}
	vendor := fmt.Sprintf("%s/%s", pwd, VendorDir)
	err = os.Setenv("GOPATH", vendor)
	if err != nil {
		log.Fatal(err)
	}
}
