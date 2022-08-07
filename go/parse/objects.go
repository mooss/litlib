package parse

import (
	"regexp"
	"strings"
)

///////////////////////////
// Object string library //
///////////////////////////

type str struct{ string }

func (p str) IsPrefix(s string) bool {
	return strings.HasPrefix(s, p.string)
}

func (p str) StripLeftOf(s string) string {
	return strings.TrimPrefix(s, p.string)
}

func (set str) Trim(s string) string {
	return strings.Trim(s, set.string)
}

func (set str) TrimRight(s string) string {
	return strings.TrimRight(s, set.string)
}

func (set str) HasRune(r rune) bool {
	for _, cr := range set.string {
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
	return strings.IndexAny(s, set.string)
}

// Intersects returns true when all runes in s are also in set.
func (set str) Intersects(s string) bool {
	// There is probably some trick to make this faster.
	for _, r := range s {
		for _, ref := range set.string {
			if ref != r {
				return false
			}
		}
	}
	return true
}

func (sep str) Join(s ...string) string {
	return strings.Join(s, sep.string)
}

var spaces = str{" \t\n"}

// IDEA: object prefix and suffix library.

///////////////////////////
// Object regexp library //
///////////////////////////

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
