package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	lenient := flag.Bool("lenient", false, "allow constructs that are technically valid per the GFM table spec but are usually mistakes")
	recursive := flag.Bool("recursive", false, "if a FILE argument is a directory, walk it recursively for .md and .markdown files")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdtable-lint [--lenient] [--recursive] FILE [FILE...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	exitCode := 0
	for _, path := range paths {
		files, err := resolvePath(path, *recursive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtable-lint: %s: %v\n", path, err)
			exitCode = 2
			continue
		}
		for _, file := range files {
			if code := lintFile(file, *lenient); code > exitCode {
				exitCode = code
			}
		}
	}

	os.Exit(exitCode)
}

// resolvePath expands path into the list of files to lint. A plain file is
// returned as-is. A directory is only expanded when recursive is set -
// silently skipping it or silently recursing into it are both surprising.
func resolvePath(path string, recursive bool) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	if !recursive {
		return nil, fmt.Errorf("is a directory (use --recursive to scan it)")
	}

	var files []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isMarkdownFile(d.Name()) {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func isMarkdownFile(name string) bool {
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown")
}

// lintFile reads and lints one file, printing findings to stdout and any I/O
// error to stderr. It returns the exit code this file alone contributes: 2 on
// an I/O error, 1 if any error-level finding was reported, 0 otherwise.
func lintFile(path string, lenient bool) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdtable-lint: %s: %v\n", path, err)
		return 2
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	code := 0
	for _, f := range LintLines(path, lines, lenient) {
		fmt.Println(f.String())
		if f.Severity == SeverityError && code < 1 {
			code = 1
		}
	}
	return code
}
