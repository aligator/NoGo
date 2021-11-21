package main

import (
	"fmt"
	"io/fs"
	"nogo"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fs.WalkDir()
	err = nogo.WalkDir(os.DirFS(wd), ".gitignore", ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path, d, err)
		return err
	})
	if err != nil {
		panic(err)
	}
}
