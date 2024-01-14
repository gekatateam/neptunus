//go:build mage

package main

import (
	"fmt"
	"os"
	"runtime"

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
		}, "go", "build", "-o", joinPath(path, fmt.Sprintf("neptunus%v", b.extention)), buildFrom); err != nil {
			return fmt.Errorf("GOOS=%v GOARCH=%v - build failed: %w", b.goos, b.goarch, err)
		}

		fmt.Printf("GOOS=%v GOARCH=%v - packing\n", b.goos, b.goarch)

		if err := archive(path, b); err != nil {
			return fmt.Errorf("GOOS=%v GOARCH=%v - packing failed: %w", b.goos, b.goarch, err)
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

func archive(path string, b buildInfo) error {
	buildVersion := version()
	binary := fmt.Sprintf("neptunus%v", b.extention)

	switch b.archive {
	case "zip":
		if runtime.GOOS == "windows" {
			return sh.Run("powershell", 
				"Compress-Archive", 
				joinPath(path, binary), 
				joinPath(".", buildDir, fmt.Sprintf("neptunus-%v-%v-%v.zip", b.goos, b.goarch, buildVersion)),
			)
		}

		return sh.Run("zip", 
			"-vj", 
			joinPath(".", buildDir, fmt.Sprintf("neptunus-%v-%v-%v.zip", b.goos, b.goarch, buildVersion)), 
			joinPath(path, binary),
		)
	case "tar":
		return sh.Run("tar", 
			"-C", 
			path, 
			"-cvzf", 
			joinPath(".", buildDir, fmt.Sprintf("neptunus-%v-%v-%v.tar.gz", b.goos, b.goarch, buildVersion)), 
			binary,
		)
	default:
		return fmt.Errorf("unexpected archive type - %v", b.archive)
	}
}
