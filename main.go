package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func goList(pkgs []string) ([]string, error) {
	args := append([]string{"list", "-e", "--"}, pkgs...)

	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not create stdout pipe to go list: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not exec go list: %v", err)
	}

	scan := bufio.NewScanner(stdout)
	var acc []string
	for scan.Scan() {
		dir := scan.Text()
		if dir[0] == '_' {
			continue
		}
		acc = append(acc, dir)
	}
	if err := scan.Err(); err != nil {
		_ = cmd.Wait()
		return nil, fmt.Errorf("could not read stdout from go list: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("could not exec go list: %v", err)
	}

	return acc, nil
}

func filesOf(dir string) ([]string, error) {
	pkg, err := build.Import(dir, "", build.FindOnly)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(pkg.Dir, "*.go"))
	if err != nil {
		return nil, err
	}
	for i := range matches {
		if r := matches[i][0]; r == '.' || r == '_' {
			matches[i] = ""
		}
	}
	return matches, nil
}

func count(file string) (Struct, TaggedStruct int) {
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		return
	}
	ast.Inspect(f, func(n ast.Node) bool {
		// Skip struct{}.
		if s, ok := n.(*ast.StructType); ok && len(s.Fields.List) > 0 {
			Struct++
			if tagged(s) {
				TaggedStruct++
			}
		}
		return true
	})

	return Struct, TaggedStruct
}

func tagged(s *ast.StructType) bool {
	for _, field := range s.Fields.List {
		if field.Tag != nil {
			return true
		}
	}
	return false
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	dirs, err := goList(flag.Args())
	if err != nil {
		if len(dirs) == 0 {
			log.Fatal(err)
		} else {
			log.Print(err)
		}
	}

	var Structs, Tagged int
	for _, dir := range dirs {
		files, err := filesOf(dir)
		if err != nil {
			log.Print(err)
			continue
		}
		for _, file := range files {
			if file == "" {
				continue
			}
			Struct, TaggedStructs := count(file)
			Structs += Struct
			Tagged += TaggedStructs
		}
	}

	fmt.Println("Tagged", Tagged)
	fmt.Println("Total", Structs)
}
