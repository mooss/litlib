# The source documents from which everything stems:
ORG_SOURCES := README cpp

.PHONY: pdf

# The following syntax allow us to add a suffix to the ORG_SOURCES list.
# This means that the org sources are to be compiled into pdf files.
pdf: $(ORG_SOURCES:%=%.pdf)

%.pdf: %.org
	./export-to-pdf.sh "$^" "$@"
