// Check tries to implement the output `git check-ignore`

package main

import (
	"flag"
	"fmt"
	"github.com/aligator/nogo"
	"io/fs"
	"os"
	"strings"
)

func main() {
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// DirFs actually implements StatFS, so we can use it.
	wdfs := os.DirFS(wd).(fs.StatFS)

	n, err := nogo.ForFS(wdfs, ".gitignore", nogo.DotGitRule)
	if err != nil {
		panic(err)
	}

	files := flag.Args()
	for _, toSearch := range files {
		toSearch = strings.TrimPrefix(toSearch, "./")
		if toSearch == "" {
			toSearch = "."
		}

		info, err := wdfs.Stat(toSearch)
		if err != nil {
			panic(err)
		}

		if info.Name() == ".git" {
			return
		}

		if err != nil {
			panic(err)
		}

		if n.Match(toSearch, info.IsDir()) {
			fmt.Printf("./%v\n", toSearch)
		}
	}
}
