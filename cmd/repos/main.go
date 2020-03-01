package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type folder struct {
	name  string
	depth int
}

type folders []folder

func (a *folders) String() string {
	return fmt.Sprintf("%+v", *a)
}

func (a *folders) Set(x string) error {
	p := strings.Split(x, ",")
	if len(p) > 2 {
		return errors.New("too many occurances of , in argument")
	}
	n := -1
	if len(p) == 2 {
		var err error
		n, err = strconv.Atoi(p[1])
		if err != nil {
			return err
		}
	}
	*a = append(*a, folder{name: p[0], depth: n})
	return nil
}

var flagFolders folders

func main() {
	flag.Var(&flagFolders, "p", "path[,depth]; may be specified multiple times.")
	flag.Parse()

	for _, folder := range flagFolders {
		for _, path := range Find(folder.name, folder.depth) {
			rel, err := filepath.Rel(folder.name, path)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s,%s\n", folder.name, rel)
		}
	}
}

// Find git repositories.
func Find(name string, depth int) []string {
	var repos []string
	filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			return nil
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(name, path)
		if err != nil {
			panic(err)
		}

		if depth == 0 && rel == "." {
			return filepath.SkipDir
		}

		if depth > 0 && rel != "." {
			p := strings.Split(rel, string(filepath.Separator))
			if len(p) >= depth {
				return filepath.SkipDir
			}
		}

		return nil
	})
	return repos
}
