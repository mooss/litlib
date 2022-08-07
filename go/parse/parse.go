package parse

import (
	"fmt"
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

///////////////
// Particles //
///////////////

// Particle represents a part of a document that has been parsed.
type Particle struct {
	ParticleImpl
}

// ParticleImpl is the interface that a type must implement to be embeddable
// into a Particle.
// This embedding step is done as a way to add convenience methods in a central
// point, thus having less things to implement in each ParticleImpl.
type ParticleImpl interface {
	// Repr returns a simple representation of the particle.
	Repr() []string
}

// Dump dumps the particle to stdout for debugging purposes.
func (p Particle) Dump() {
	fmt.Printf("%T {\n", p.ParticleImpl)
	for _, token := range p.Repr() {
		fmt.Printf("    ^%s$\n", token)
	}
	fmt.Println("}")
}

// void returns true if this Particle holds no implementation.
func (p Particle) void() bool {
	return p.ParticleImpl == nil
}

// Particles is a sequence of parsed Particle.
type Particles []Particle

// Dump dumps all the contained Particles to stdout for debugging pusposes.
func (ps Particles) Dump() {
	for _, p := range ps {
		p.Dump()
	}
}

// Parameters is metadata attached to a file or a particle.
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

// CodeParticle represents code, content meant for machine consumption.
type CodeParticle struct {
	Raw    []string   // Code.
	Lang   string     // Identifier of the language.
	Params Parameters // Parameters of the code block.
}

func (c CodeParticle) Repr() []string {
	return c.Raw
}

// ProseParticle represents prose, content meant for human consumption.
type ProseParticle struct {
	Raw []string // Unparsed lines, with Markup particles left as-is.
}

func (p ProseParticle) Repr() []string {
	return p.Raw
}

// MetadataParticle holds metadata about the document.
type MetadataParticle struct {
	Name     string
	RawValue string
}

func (m MetadataParticle) Repr() []string {
	return []string{m.Name + "=" + m.RawValue}
}

// SectionParticle represents a section marker, symbolising a new branch of the
// document tree.
type SectionParticle struct {
	Title string
	Level int
}

func (m SectionParticle) Repr() []string {
	return []string{"level=" + fmt.Sprint(m.Level) + ", title=" + m.Title}
}

// SpaceParticle represents whitespace (space, newline and tab), content that
// typically holds no particular meaning within a document.
type SpaceParticle struct {
	Raw []string
}

func (v SpaceParticle) Repr() []string {
	return v.Raw
}

////////////////////////
// Parsing primitives //
////////////////////////

type Taker func([]string) int
type Baker func(string) string
type Maker func([]string) ParticleImpl

// Atom is the smallest parsing entity.
// It defines how to produce a given particle from raw text.
type Atom struct {
	Take Taker // How many lines to take.
	Bake Baker // How to transform a single line.
	Make Maker // How to make a particle with transformed lines.
	// IDEA: MonoTake, MonoMake for more convenient definition of one line particles.
	// IDEA: ErrorTake, ErrorMake to get explanations on why parsing failed.
	//       Could provide useful error diagnostics.
}

// Emit tries to parse the given lines, returning the lines that were not taken
// as well as the Particle that was made.
// When the lines are not parsed, a void Particle is emmited.
// The error is always nil but it will be used at some point as a mechanism to
// provide error diagnostics.
func (a Atom) Emit(lines []string) ([]string, Particle, error) {
	take := a.Take(lines)
	if take == 0 {
		return lines, Particle{}, nil
	}
	return lines[take:], Particle{a.Make(Map(a.Bake, lines[:take]))}, nil
}

// Molecule is a group of Atom defining all the necessary logic to parse a
// literate document.
type Molecule []Atom

// Parse tries to parse the given lines with its Atoms.
// If several of its Atoms can parse a given line, the first one is chosen,
// hence to correctly parse a document, it is primordial to pay attention to the
// order of the Atoms.
func (m Molecule) Parse(lines []string) (Particles, error) {
	res := Particles{}
	for len(lines) > 0 {
		var emitted Particle
		var err error
		for _, atom := range m {
			lines, emitted, err = atom.Emit(lines)
			if err != nil {
				return nil, err
			}
			if !emitted.void() { // Managed to find an atom parsing the lines.
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

// GreedyTake builds a Taker function that will take all the consecutive
// lines that satisfies its predicate.
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

////////////////////////////
// Other atomic functions //
////////////////////////////

// ReSectionMake generates a section Maker with a regexp that produces two groups:
//  - The section specifier whose length is the level.
//  - The section title.
// The correctness of the regex is of course the responsibility of the caller.
func ReSectionMake(r regex) Maker {
	return func(lines []string) ParticleImpl {
		groups := r.Groups(lines[0])
		return SectionParticle{Level: len(groups[1]), Title: groups[2]}
	}
}

// NoBk returns its raw argument.
func NoBk(l string) string { return l }

// ProseMk makes a ProseParticle.
func ProseMk(ls []string) ParticleImpl { return ProseParticle{ls} }

// SpaceMk makes a SpaceParticle.
// It is the responsibility of the caller to ensure that its argument is indeed
// whitespace.
func SpaceMk(ls []string) ParticleImpl { return SpaceParticle{Raw: ls} }

////////////////////
// Reusable atoms //
////////////////////

// SpaceAtom takes lines composed exclusively of whitespace.
var SpaceAtom = Atom{
	Take: GreedyTake(spaces.Intersects),
	Bake: NoBk,
	Make: SpaceMk,
}

///////////////
// Languages //
///////////////

// Fuser represents a function able to fuse particles together to form a
// document.
// Fusing is therefore the dual of parsing.
type Fuser func(Particles) ([]string, error)

// Language represents a language, be it prose-based or code-based, and all that
// is needed to manipulate it.
type Language struct {
	Identifiers []string
	Extensions  []string
	Parser      Molecule
	Fuse        Fuser
}

func (l Language) Parse(lines []string) (Particles, error) {
	return l.Parser.Parse(lines)
}
