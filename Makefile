# The source documents from which everything stems:
ORG_SOURCES := README cpp

.PHONY: pdf

# The following syntax allow us to add a suffix to the ORG_SOURCES list.
# This means that the org sources are to be compiled into pdf files.
pdf: $(ORG_SOURCES:%=%.pdf)

%.pdf: %.org
	pandoc 	--standalone\
		--to latex $^\
		--output $@\
		--include-in-header preamble.tex\
		--highlight-style tango\
		--variable documentclass:report\
		--variable geometry:margin=2.5cm\
		--variable papersize:a4paper\
		--table-of-content\
		--toc-depth 5\
		--citeproc\
		--variable lang:en
