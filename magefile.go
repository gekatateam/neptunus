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
	buildFrom := joinPath(".", "cmd", "neptunus")
	buildVersion := version()

	for _, b := range builds {
		path := joinPath(".", buildDir, fmt.Sprintf("%v-%v-%v", b.goos, b.goarch, buildVersion))
		os.Mkdir(path, os.ModePerm)

		fmt.Printf("GOOS=%v GOARCH=%v - building\n", b.goos, b.goarch)

		if err := sh.RunWith(map[string]string{
			"GOOS":   b.goos,
			"GOARCH": b.goarch,
		}, "go", "build", "-o", joinPath(path, fmt.Sprintf("neptunus%v", b.extention)), buildFrom); err != nil { // path + s + fmt.Sprintf("neptunus%v", b.extention), "." + s + "cmd" + s + "neptunus" + s)
			return fmt.Errorf("GOOS=%v GOARCH=%v - build failed: %w", b.goos, b.goarch, err)
		}
	}

	return nil
}

func Docker() error {
	return nil
}

func Clear() error {
	return os.RemoveAll(joinPath(".", buildDir))
}

func version() string {
	return "0.1.0"
}

func joinPath(elem... string) string {
	var s = string(os.PathSeparator)
	var r = ""

	for i, e := range elem {
		r += e
		if i == len(elem)-1 {
			break
		}
		r += s
	}

	return r
}
