package main

import (
	"flag"
	"log"
	"os"
)

var file = flag.String("file", "", "Path to template file to compile. Compile template file is placed near the original file "+
	"with .go extension")

func main() {
	flag.Parse()

	infile := *file
	inf, err := os.Open(infile)
	if err != nil {
		log.Fatalf("cannot open file %q: %s", infile, err)
	}
	defer inf.Close()

	outfile := infile + ".go"
	outf, err := os.Create(outfile)
	if err != nil {
		log.Fatalf("cannot create file %q: %s", outfile, err)
	}
	defer outf.Close()

	packageName, err := getPackageName(infile)
	if err != nil {
		log.Fatalf("cannot determine package name for %q: %s", infile, err)
	}
	if err = parse(outf, inf, infile, packageName); err != nil {
		log.Fatalf("error when parsing file %q: %s", infile, err)
	}
}
