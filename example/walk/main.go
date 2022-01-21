package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/aligator/nogo"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	walkFS := os.DirFS(wd)
	n := nogo.New(nogo.DotGitRule)

	// First load the ignore files in the fs.
	if err := n.AddFromFS(walkFS, ".gitignore"); err != nil {
		panic(err)
	}

	// And then use nogo to walk by ignoring ignored files / folders.
	err = fs.WalkDir(n.ForWalkDir(walkFS, ".", func(path string, d fs.DirEntry, err error) error {
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
