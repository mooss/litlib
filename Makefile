# The source documents from which everything stems:
ORG_SOURCES := README cpp pandoc

.PHONY: pdf LitLib

# The following syntax allow us to add a suffix to the ORG_SOURCES list.
# This means that the org sources are to be compiled into pdf files.
pdf: $(ORG_SOURCES:%=%.pdf)

%.pdf: %.org
	./export-to-pdf.sh "$^" "$@"

# TODO: Define a filter list to make sure all filters are generated.
filters: pandoc.org
	./include.pl $< ':tangle'

# This target gathers all the targets that are supposed to be useful from the outside.
LitLib: filters
