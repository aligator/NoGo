package main

import (
	"fmt"
	"github.com/aligator/nogo"
	"io/fs"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	walkFS := os.DirFS(wd)
	n, err := nogo.ForFS(walkFS, ".gitignore", nogo.DotGitRule)
	if err != nil {
		panic(err)
	}

	err = fs.WalkDir(walkFS, ".", n.WalkDirFunc(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path, d.Name())
		return nil
	}))

	if err != nil {
		panic(err)
	}
}
