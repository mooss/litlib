#!/usr/bin/env bash

# Will be used to fetch the correct resources no matter where the script is called
# from.
where=$(dirname "$0")

if [ $# -ne 2 ]
then
    echo "Usage: $0 source destination.pdf"
    exit 1
fi

source=$1
destination=$2

pandoc --standalone\
       --to latex $source\
       --output $destination\
       --include-in-header $where/preamble.tex\
       --highlight-style tango\
       --variable documentclass:report\
       --variable geometry:margin=2.5cm\
       --variable papersize:a4paper\
       --table-of-content\
       --toc-depth 5\
       --citeproc\
       --variable lang:en
