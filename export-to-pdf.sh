#!/usr/bin/env bash

# Will be used to fetch the correct resources no matter where the script is called
# from.
where=$(dirname "$0")

if [ $# -ne 2 ]
then
    echo "Usage: $0 source destination.pdf"
    exit 1
fi

source="$1"
destination="$2"

function process_source() {
    cat "$source"
}

process_source | pandoc --standalone\
                        --from org\
                        --to latex\
                        --output "$destination"\
                        --include-in-header "$where/preamble.tex"\
                        --highlight-style tango\
                        --variable documentclass:report\
                        --variable geometry:margin=2.5cm\
                        --variable papersize:a4paper\
                        --table-of-content\
                        --toc-depth 5\
                        --citeproc\
                        --variable lang:en\
                        --lua-filter "$where/filters/minipage.lua"\
                        --lua-filter "$where/filters/noweb-call.lua"\
                        --lua-filter "$where/filters/comment-noweb-in-bash.lua"\
