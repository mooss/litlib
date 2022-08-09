package parse

import (
	"strings"
)

func NextNowebKeyValues(data string) (key string, values []string, rest string) {
	skimspace := func() int {
		for i, r := range rest {
			if r != ' ' {
				return i
			}
		}
		return -1
	}

	slicevals := func(s string) []string {
		return strings.Fields(strings.TrimRight(s, " "))
	}

	/////////////////////
	// Extract the key //
	rest = data
	idx := skimspace()
	if idx == -1 {
		return "", nil, "" // There are only spaces.
	}
	rest = data[idx:]
	if rest[0] == ':' {
		end := strings.IndexByte(rest[1:], ' ') + 1
		if end == 0 {
			return rest[1:], nil, "" // key without value.
		}
		key = rest[1:end]
		rest = rest[end+1:]
	} // key remains the empty string.

	////////////////////////
	// Extract the values //
	idx = skimspace()
	if idx == -1 {
		return key, nil, "" // key without value.
	}
	rest = rest[idx:]

	end := strings.Index(rest, " :")
	if end == -1 {
		return key, slicevals(rest), "" // Last value.
	}
	return key, slicevals(rest[:end]), rest[end+1:]
}

// ParseNowebArguments parses noweb arguments into an argument map.
// For example, ":exports none :include iostream vector :minipage" becomes:
// map[string][]string {
//     "exports": ["none"],
//     "include": ["iostream", "vector"],
//     "minipage": [],
// }
func ParseNowebArguments(source string) Parameters {
	res := Parameters{}
	var key string
	var values []string

	for len(source) > 0 {
		key, values, source = NextNowebKeyValues(source)
		if key != "" || len(values) > 0 {
			res.Add(key, values)
		}
	}
	return res
}
