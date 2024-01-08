//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

type buildInfo struct{
	goos      string
	goarch    string
	extention string
	archive   string
}

var builds = []buildInfo{
	{"linux",   "amd64", "",     "tar"},
	{"windows", "amd64", ".exe", "zip"},
}

var buildDir = "builds"
var s = string(os.PathSeparator)

func Build() error {
	for _, b := range builds {
		path := "." + s + buildDir + s + fmt.Sprintf("%v-%v-%v", b.goos, b.goarch, version())
		os.Mkdir(path, os.ModePerm)

		if err := sh.RunWith(map[string]string{
			"GOOS": b.goos,
			"GOARCH": b.goarch,
		}, "go", "build", "-o", path + s + fmt.Sprintf("neptunus%v", b.extention), "." + s + "cmd" + s + "neptunus" + s); err != nil {
			return fmt.Errorf("GOOS=%v GOARCH=%v build failed: %w", b.goos, b.goarch, err)
		}
	}

	return nil
}

func Docker() error {
	return nil
}

func Clear() error {
	return os.RemoveAll("." + s + buildDir)
}

func version() string {
	return "0.1.0"
}
