package main

import (
	"encoding/json"
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
	format := flag.String("format", "text", "output format: text or json")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdtable-lint [--lenient] [--recursive] [--format text|json] FILE [FILE...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "mdtable-lint: invalid --format %q (want text or json)\n", *format)
		os.Exit(2)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	exitCode := 0
	var allFindings []Finding
	for _, path := range paths {
		files, err := resolvePath(path, *recursive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtable-lint: %s: %v\n", path, err)
			exitCode = 2
			continue
		}
		for _, file := range files {
			findings, err := lintFile(file, *lenient)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mdtable-lint: %s: %v\n", file, err)
				if exitCode < 2 {
					exitCode = 2
				}
				continue
			}
			for _, f := range findings {
				if f.Severity == SeverityError && exitCode < 1 {
					exitCode = 1
				}
			}
			if *format == "text" {
				for _, f := range findings {
					fmt.Println(f.String())
				}
			} else {
				allFindings = append(allFindings, findings...)
			}
		}
	}

	if *format == "json" {
		if allFindings == nil {
			allFindings = []Finding{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(allFindings); err != nil {
			fmt.Fprintf(os.Stderr, "mdtable-lint: %v\n", err)
			os.Exit(2)
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

// lintFile reads and lints one file, returning its findings. An I/O error
// reading the file is returned rather than printed, so callers can format it
// consistently with the rest of their output.
func lintFile(path string, lenient bool) ([]Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	return LintLines(path, lines, lenient), nil
}
