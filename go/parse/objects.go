package parse

import (
	"regexp"
	"strings"
)

///////////////////////////
// Object string library //
///////////////////////////

type str string

func (p str) IsPrefix(s string) bool {
	return strings.HasPrefix(s, string(p))
}

func (p str) StripLeftOf(s string) string {
	return strings.TrimPrefix(s, string(p))
}

func (set str) Trim(s string) string {
	return strings.Trim(s, string(set))
}

func (set str) TrimRight(s string) string {
	return strings.TrimRight(s, string(set))
}

func (set str) HasRune(r rune) bool {
	for _, cr := range string(set) {
		if cr == r {
			return true
		}
	}
	return false
}

func (set str) Fields(s string) []string {
	return strings.FieldsFunc(s, set.HasRune)
}

func (set str) First(s string) int {
	return strings.IndexAny(s, string(set))
}

// Skim returns the first index of s that is not in the set.
// Returns -1 when the set intersects with s.
func (set str) Skim(s string) int {
	// There is probably some trick to make this faster.
	for i, r := range s {
		found := true
		for _, ref := range string(set) {
			if ref == r {
				found = false
			}
		}
		if found {
			return i
		}
	}
	return -1
}

// Intersects returns true when all runes in s are also in set.
func (set str) Intersects(s string) bool {
	return set.Skim(s) == -1
}

func (sep str) Join(s ...string) string {
	return strings.Join(s, string(sep))
}

var spaces = str(" \t\n")

// IDEA: object prefix and suffix library.

///////////////////////////
// Object regexp library //
///////////////////////////

// Unlike str, regex embeds the type it extends.
// This is because I'm not sure of the implications of converting to the
// underlying Regexp and taking a pointer to it.
// Furthermore, unlike str I see no special interest in using a type definition.

type regex struct{ *regexp.Regexp }

func re(s string) regex                  { return regex{regexp.MustCompile(s)} }
func (r regex) Match(s string) bool      { return r.MatchString(s) }
func (r regex) Groups(s string) []string { return r.FindStringSubmatch(s) }

//////////////////////////
// Object slice library //
//////////////////////////

type slice[T any] []T

func slc[T any](el ...T) slice[T] {
	return slice[T](el)
}

func pslc[T any](el ...T) *slice[T] {
	res := slice[T](el)
	return &res
}

// Add adds element at the end of the slice, returning a pointer to the slice.
func (s *slice[T]) Add(el ...T) *slice[T] {
	*s = append(*s, el...)
	return s
}
