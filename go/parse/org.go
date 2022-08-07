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
var orgBeginPfx = str{"#+begin_"}
var orgEndPfx = str{"#+end_"}

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

// OrgBlockMk makes a block particle from Org lines.
func OrgBlockMk(lines []string) ParticleImpl {
	return BlockParticle{
		Raw:  lines[1 : len(lines)-1],
		Type: orgBeginPfx.StripLeftOf(lines[0]),
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
	Atom{ // Other kind of blocks, like quote blocks.
		// This taker doesn't ensure that the begin and end block are matching.
		// It will work fine assuming no wild ^#+end_ is present inside blocks.
		// This is bound to happen eventually so I guess this is a TODO.
		Take: BetweenTake(orgBeginPfx.IsPrefix, orgEndPfx.IsPrefix),
		Bake: NoBk,
		Make: OrgBlockMk,
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
				begin += " " + p.Params.FuseToNoweb()
			}
			res.Add(begin)
			res.Add(p.Raw...)
			res.Add(orgEndSrcPfx.string)

		case ProseParticle:
			res.Add(p.Raw...)

		case MetadataParticle:
			prop := "#+" + p.Name + ":"
			if p.RawValue != "" {
				prop += p.RawValue
			}
			res.Add(prop)

		case SectionParticle:
			res.Add(strings.Repeat("*", p.Level) + " " + p.Title)

		case SpaceParticle:
			res.Add(p.Raw...)

		case BlockParticle:
			res.Add(orgBeginPfx.string + p.Type)
			res.Add(p.Raw...)
			res.Add(orgEndPfx.string + p.Type)

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
