package parse

import (
	"fmt"
	"strings"
)

///////////////////
// Text matching //
///////////////////

var orgSectionRe = re(`^(\*+) (.+)$`)
var orgBeginSrcPfx = str{"#+begin_src"}
var orgEndSrcPfx = str{"#+end_src"}
var orgPropertyPfx = str{"#+"}

////////////
// Makers //
////////////

// OrgCodeMk makes a code particle from Org lines.
func OrgCodeMk(lines []string) ParticleImpl {
	lang, params := ParseOrgBeginSrc(lines[0])
	return CodeParticle{
		Raw:    lines[1 : len(lines)-1],
		Lang:   lang,
		Params: params,
	}
}

// OrgPropertyMk makes a metadata particle from an Org property line.
func OrgPropertyMk(lines []string) ParticleImpl {
	line := lines[0]
	split := strings.SplitN(line, ":", 2)
	res := MetadataParticle{Name: spaces.Trim(split[0])}
	if len(split) == 2 {
		res.RawValue = spaces.TrimRight(split[1])
	}
	return res
}

///////////////////////////////////
// High-level parsing and fusing //
///////////////////////////////////

// OrgMolecule is a sequence of atomic parsers able to parse an Org file.
var OrgMolecule = Molecule{
	Atom{ // Section, hierarchical delimiter of the document.
		Take: FirstTake(orgSectionRe.Match),
		Bake: NoBk,
		Make: ReSectionMake(orgSectionRe),
	},
	Atom{ // Code, content meant for machine consumption.
		Take: BetweenTake(orgBeginSrcPfx.IsPrefix, orgEndSrcPfx.IsPrefix),
		Bake: NoBk,
		Make: OrgCodeMk,
	},
	Atom{ // Metadata about the document.
		Take: FirstTake(orgPropertyPfx.IsPrefix),
		Bake: orgPropertyPfx.StripLeftOf,
		Make: OrgPropertyMk,
	},
	SpaceAtom, // Whitespace, content that can typically be ignored.
	Atom{ // Prose, content meant for human consumption.
		Take: TrailingTake(spaces.Intersects, nor(orgSectionRe.Match, orgPropertyPfx.IsPrefix)),
		Bake: NoBk,
		Make: ProseMk,
	},
}

// OrgFuser can reconstruct the lines of an Org document from parsed particles.
func OrgFuser(matter Particles) ([]string, error) {
	res := slice[string]{}
	for _, part := range matter {
		switch p := part.ParticleImpl.(type) {
		case CodeParticle:
			begin := orgBeginSrcPfx.string + " " + p.Lang
			if len(p.Params) > 0 {
				begin += " " + FuseNowebArguments(p.Params)
			}
			res.add(begin)
			res.add(p.Raw...)
			res.add(orgEndSrcPfx.string)

		case ProseParticle:
			res.add(p.Raw...)

		case MetadataParticle:
			prop := "#+" + p.Name
			if p.RawValue != "" {
				prop += ":" + p.RawValue
			}
			res.add(prop)

		case SectionParticle:
			res.add(strings.Repeat("*", p.Level) + " " + p.Title)

		case SpaceParticle:
			res.add(p.Raw...)

		default:
			return nil, fmt.Errorf("no org fuser for %T", part.ParticleImpl)
		}
	}
	return res, nil
}

// OrgLang holds information needed to manipulate Org files.
var OrgLang = Language{
	Identifiers: []string{"org"},
	Extensions:  []string{".org"},
	Parser:      OrgMolecule,
	Fuse:        OrgFuser,
}

///////////
// Noweb //
///////////

// ParseNowebArguments parses noweb arguments into an argument map.
// For example, ":exports none :include iostream vector :minipage" becomes:
// map[string][]string {
//     "exports": ["none"],
//     "include": ["iostream", "vector"],
//     "minipage": [],
// }
func ParseNowebArguments(args string) Parameters {
	args = spaces.Trim(args)
	defs := strings.Split(args, ":")

	res := Parameters{}
	if !strings.HasPrefix(args, ":") {
		res.Add("", spaces.Fields(defs[0]))
	}
	defs = defs[1:]

	for _, argspec := range defs {
		fields := spaces.Fields(argspec)
		res.Add(fields[0], fields[1:])
	}

	return res
}

// FuseNowebArguments transforms parameters into a noweb string.
func FuseNowebArguments(ps Parameters) string {
	return strings.Join(Map(func(p Parameter) string {
		res := ":" + p.Key
		if len(p.Values) > 0 {
			res += " " + strings.Join(p.Values, " ")
		}
		return res
	}, ps), " ")
}

// ParseOrgBeginSrc parses the language and noweb parameters of a `#+begin_src`
// line.
func ParseOrgBeginSrc(line string) (string, Parameters) {
	line = orgBeginSrcPfx.StripLeftOf(line)
	line = spaces.Trim(line)
	pos := spaces.First(line)
	if pos == -1 {
		return line, Parameters{}
	}
	return line[:pos], ParseNowebArguments(line[pos:])
}
