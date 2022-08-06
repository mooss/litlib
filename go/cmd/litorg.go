package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mooss/litlib/parse"
)

func exit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(0)
}

func nofail(err error) {
	if err != nil {
		exit(err.Error())
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		exit(fmt.Sprint("Usage: ", os.Args[0], " filename"))
	}

	filename := flag.Arg(0)
	content, err := ioutil.ReadFile(filename)
	nofail(err)

	parser := parse.OrgLang.Parser
	parsed, err := parser.Parse(strings.Split(string(content), "\n"))
	nofail(err)

	parsed.Dump()
}
