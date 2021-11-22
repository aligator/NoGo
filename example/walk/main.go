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

	err = nogo.WalkDir([]string{".gitignore"}, os.DirFS(wd), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path, d.Name())
		return nil
	}, nogo.WithRules(nogo.GitIgnoreRule...))

	if err != nil {
		panic(err)
	}
}
