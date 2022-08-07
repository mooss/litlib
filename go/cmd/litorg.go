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

	parsed, err := parse.OrgLang.Parse(strings.Split(string(content), "\n"))
	nofail(err)

	fused, err := parse.OrgLang.Fuse(parsed)
	nofail(err)

	fmt.Println(strings.Join(fused, "\n"))
}
