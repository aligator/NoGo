# NoGo [![test](https://github.com/aligator/nogo/actions/workflows/test.yaml/badge.svg)](https://github.com/aligator/nogo/actions/workflows/test.yaml) [![CodeQL](https://github.com/aligator/nogo/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/aligator/nogo/actions/workflows/codeql-analysis.yml)
A .gitignore parser for Go.

## Features
* parsing .gitignore files
* loading file trees with several .gitignore files
* fs.WalkDir WalkDirFunc implementation (and afero.Walk (see below))
* customizable ignore filename (instead of .gitignore)
* full compatibility with git  
As far as I could test it, it handles .gitignore files the same way as git.  
If you find an inconsistency with git, please create a new Issue.  
The goal is to provide the exact same .gitignore handling.

## Usage
```go
n := nogo.New(nogo.DotGitRule)
if err := n.AddFromFS(wdfs, ".gitignore"); err != nil {
    panic(err)
}

match := n.Match(toSearch, isDir)
fmt.Println(match)
```

There is also an alternative MatchBecause method which returns also
the causing rule if you need some context.

There exists a predefined rule to ignore any `.git` folder automatically.
```go
n := nogo.New(nogo.DotGitRule)
if err := n.AddFromFS(wdfs, ".gitignore"); err != nil {
    panic(err)
}
```

## Walk
NoGo can be used with fs.WalkDir. [Just see the example walk.](example/walk/main.go)
If you need to use another Walk function, you can build your own wrapper using 
the `NoGo.WalkFunc` function. 

Example for afero:
I intentionally did not include this to avoid a new dependency
just because of afero-compatibility. However, as you can see, it is not that 
hard to build your own:
```go
package main

import (
	"fmt"
	"github.com/aligator/nogo"
	"github.com/spf13/afero"
	"io/fs"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	baseFS := afero.NewBasePathFs(afero.NewOsFs(), wd)

	n := nogo.New(nogo.DotGitRule)

	// Important: The NoGo instance (n) is NOT IMMUTABLE!
	// While walking the tree it automatically loads all found .gitignore files
	// and parses the rules of it.
	//
	// -> After running the Walk, 'n' contains the same 
	// rules as if you had used CompileAll().
	err = afero.Walk(baseFS, ".", func(path string, info fs.FileInfo, err error) error {
		if ok, err := n.WalkFunc(afero.NewIOFS(baseFS), ".gitignore", path, info.IsDir(), err); !ok {
			return err
		}

		fmt.Println(path, info.Name())
		return nil
	})

	if err != nil {
		panic(err)
	}
}
```