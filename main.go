package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	lenient := flag.Bool("lenient", false, "allow constructs that are technically valid per the GFM table spec but are usually mistakes")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdtable-lint [--lenient] FILE [FILE...]")
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
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtable-lint: %s: %v\n", path, err)
			exitCode = 2
			continue
		}

		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		for _, f := range LintLines(path, lines, *lenient) {
			fmt.Println(f.String())
			if f.Severity == SeverityError && exitCode < 1 {
				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}
