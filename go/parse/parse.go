package parse

import (
	"fmt"
	"strings"
)

////////////////////////
// Functional helpers //
////////////////////////

// Pred is a predicate for type T.
type Pred[T any] func(T) bool

// and combines the given predicates into one, using the and logical operator.
func and[T any](preds ...Pred[T]) Pred[T] {
	return func(t T) bool {
		for _, p := range preds {
			if !p(t) {
				return false
			}
		}
		return true
	}
}

// nor combines the given predicates into one, using the nor logical operator.
func nor[T any](preds ...Pred[T]) Pred[T] {
	return func(t T) bool {
		for _, p := range preds {
			if p(t) {
				return false
			}
		}
		return true
	}
}

func Map[T, U any](fun func(T) U, source []T) []U {
	res := make([]U, len(source))
	for i, el := range source {
		res[i] = fun(el)
	}
	return res
}

//////////////
// Elements //
//////////////

// Element represents a part of a document that has been parsed.
type Element struct {
	ElementImpl
}

// ElementImpl is the interface that a type must implement to be embeddable into
// an Element.
// This embedding step is done as a way to add convenience methods in a central
// point, thus having less things to implement in each ElementImpl.
type ElementImpl interface {
	// Repr returns a simple representation of the element.
	Repr() []string
}

// Dump dumps the Element to stdout for debugging purposes.
func (p Element) Dump() {
	fmt.Printf("%T {\n", p.ElementImpl)
	for _, token := range p.Repr() {
		fmt.Printf("    ^%s$\n", token)
	}
	fmt.Println("}")
}

// void returns true if this Element holds no implementation.
func (p Element) void() bool {
	return p.ElementImpl == nil
}

// Elements is a sequence of parsed Element.
type Elements []Element

// Dump dumps all the contained Elements to stdout for debugging purposes.
func (ps Elements) Dump() {
	for _, p := range ps {
		p.Dump()
	}
}

// Parameters represents metadata attached to a file or a element.
// It is implemented as key-value pairs and not as a map in order to maintain
// the order.
// If performance becomes an issue, look into ordered map libraries.
type Parameters []Parameter

type Parameter struct {
	Key    string
	Values Values
}

// Values stores the values of a parameter.
// Since a typical usage would result in less than 10 values, a slice of strings
// is a simple and somewhat universal format.
type Values []string

// Has returns true if the given key is contained in the parameters.
func (ps Parameters) Has(key string) bool {
	return ps.Get(key) != nil
}

func (ps Parameters) Get(key string) *Values {
	for _, p := range ps {
		if p.Key == key {
			return &p.Values
		}
	}
	return nil
}

// Add adds values to the given key, creating it if necessary.
func (ps *Parameters) Add(key string, values Values) {
	vp := ps.Get(key)
	if vp == nil {
		*ps = append(*ps, Parameter{key, values})
	} else {
		*vp = append(*vp, values...)
	}
}

// FuseToNoweb fuses (aka serialises) parameters into noweb arguments.
func (ps Parameters) FuseToNoweb() string {
	return strings.Join(Map(func(p Parameter) string {
		res := ":" + p.Key
		if len(p.Values) > 0 {
			res += " " + strings.Join(p.Values, " ")
		}
		return res
	}, ps), " ")
}

// RawElement is not a real element but simply a shortcut to define elements
// that hold nothing more than lines of text.
// The type parameter T is just a shameful trick to generate aliases that are of
// a different type but still benefit from the methods of the aliased type.
type RawElement[T any] struct {
	Raw []string // Block content.
}

func (c RawElement[T]) Repr() []string {
	return c.Raw
}

// ProseElement represents prose, content meant for human consumption.
type ProseElement = RawElement[prose]
type prose struct{}

// SpaceElement represents whitespace (space, newline and tab), content that
// typically holds no particular meaning within a document.
type SpaceElement = RawElement[space]
type space struct{}

// BlockElement represents a special block qualified by its type.
type BlockElement struct {
	Raw  []string
	Type string
}

func (b BlockElement) Repr() []string {
	return *pslc("type=" + b.Type).Add(b.Raw...)
}

// CodeElement represents code, content meant for machine consumption.
type CodeElement struct {
	Raw    []string   // Code.
	Lang   string     // Identifier of the language.
	Params Parameters // Parameters of the code block.
}

func (c CodeElement) Repr() []string {
	return *pslc("lang="+c.Lang, "Params="+c.Params.FuseToNoweb()).Add(c.Raw...)
}

// MetadataElement holds metadata about the document.
type MetadataElement struct {
	Name     string
	RawValue string
}

func (m MetadataElement) Repr() []string {
	return slc(m.Name + "=" + m.RawValue)
}

// SectionElement represents a section marker, symbolising a new branch of the
// document tree.
type SectionElement struct {
	Title string
	Level int
}

func (m SectionElement) Repr() []string {
	return slc("level=" + fmt.Sprint(m.Level) + ", title=" + m.Title)
}

////////////////////////
// Parsing primitives //
////////////////////////

type Taker func([]string) int
type Baker func(string) string
type Maker func([]string) ElementImpl

// Rule is the smallest parsing entity.
// It defines how to produce a given element from raw text.
type Rule struct {
	Take Taker // How many lines to take.
	Bake Baker // How to transform a single line.
	Make Maker // How to make a element with transformed lines.
	// IDEA: MonoTake, MonoMake for more convenient definition of one line elements.
	// IDEA: ErrorTake, ErrorMake to get explanations on why parsing failed.
	//       Could provide useful error diagnostics.
}

// Emit tries to parse the given lines, returning the lines that were not taken
// as well as the Element that was made.
// When the lines are not parsed, a void Element is emitted.
// The error is always nil but it will be used at some point as a mechanism to
// provide error diagnostics.
func (a Rule) Emit(lines []string) ([]string, Element, error) {
	take := a.Take(lines)
	if take == 0 {
		return lines, Element{}, nil
	}
	return lines[take:], Element{a.Make(Map(a.Bake, lines[:take]))}, nil
}

// Rules represents a sequence of Rule defining all the logic necessary to parse
// a literate document.
type Rules []Rule

// Parse tries to parse the given lines with its Rules.
// If several of its Rules can parse a given line, the first one is chosen,
// hence to correctly parse a document, it is primordial to pay attention to the
// order of the Rules.
func (m Rules) Parse(lines []string) (Elements, error) {
	res := Elements{}
	for len(lines) > 0 {
		var emitted Element
		var err error
		for _, rule := range m {
			lines, emitted, err = rule.Emit(lines)
			if err != nil {
				return nil, err
			}
			if !emitted.void() { // Managed to find an rule parsing the lines.
				res = append(res, emitted)
				break
			}
		}
		if emitted.void() {
			return nil, fmt.Errorf("could not parse line `%s`", lines[0])
		}
	}
	return res, nil
}

//////////////////////
// Taker generators //
//////////////////////
// i.e. functions returning a Taker.

// GreedyTake builds a Taker function that will take all the consecutive lines
// that satisfy its predicate.
func GreedyTake(pred Pred[string]) Taker {
	return func(lines []string) int {
		for i, line := range lines {
			if !pred(line) {
				return i
			}
		}
		return len(lines)
	}
}

// FirstTake builds a Taker function that will take only one line when the
// predicate is satisfied, none otherwise.
func FirstTake(pred Pred[string]) Taker {
	return func(lines []string) int {
		if pred(lines[0]) {
			return 1
		}
		return 0
	}
}

// BetweenTake builds a Taker function that will take all the lines between its
// first and last predicates, first and last line included.
func BetweenTake(first, last Pred[string]) Taker {
	return func(lines []string) int {
		if !first(lines[0]) {
			return 0
		}
		for i, line := range lines[1:] {
			if last(line) {
				return i + 2 // Include begin and end lines.
			}
		}
		return 0 // TODO: Make taker return an error to signal that parsing went wrong?
	}
}

// TrailingTake builds a taker from two string predicates:
//  - maybe describes lines that should be taken, but not as the last line.
//  - otherwise describes lines that should always be taken.
// If a line matches both maybe and otherwise, it is treated as maybe.
func TrailingTake(maybe, otherwise Pred[string]) Taker {
	return func(lines []string) int {
		lastCore := -1
		for i, line := range lines {
			if !maybe(line) {
				if otherwise(line) {
					lastCore = i
				} else {
					break
				}
			}
		}
		return lastCore + 1
	}
}

///////////////////////
// Makers and bakers //
///////////////////////
// ReSectionMake generates a section Maker with a regexp that produces two groups:
//  - The section specifier whose length is the level.
//  - The section title.
// The correctness of the regex is of course the responsibility of the caller.
func ReSectionMake(r regex) Maker {
	return func(lines []string) ElementImpl {
		groups := r.Groups(lines[0])
		return SectionElement{Level: len(groups[1]), Title: groups[2]}
	}
}

// NoBk returns its raw argument.
func NoBk(l string) string { return l }

// ProseMk makes a ProseElement.
func ProseMk(ls []string) ElementImpl { return ProseElement{ls} }

// SpaceMk makes a SpaceElement.
// It is the responsibility of the caller to ensure that its argument is indeed
// whitespace.
func SpaceMk(ls []string) ElementImpl { return SpaceElement{Raw: ls} }

////////////////////
// Reusable rules //
////////////////////

// SpaceRule parses lines composed exclusively of whitespace.
var SpaceRule = Rule{
	Take: GreedyTake(spaces.Intersects),
	Bake: NoBk,
	Make: SpaceMk,
}

///////////////
// Languages //
///////////////

// Fuser represents a function able to fuse elements together to form a
// document.
// Fusing is therefore the dual of parsing.
type Fuser func(Elements) ([]string, error)

// Language represents a language, be it prose-based or code-based, and all that
// is needed to manipulate it.
type Language struct {
	Identifiers []string
	Extensions  []string
	Parser      Rules
	Fuse        Fuser
}

func (l Language) Parse(lines []string) (Elements, error) {
	return l.Parser.Parse(lines)
}
