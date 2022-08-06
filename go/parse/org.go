package parse

import "strings"

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
		res.RawValue = spaces.Trim(split[1])
	}
	return res
}

////////////
// Parser //
////////////

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

// OrgLang holds information needed to manipulate Org files.
var OrgLang = Language{
	Identifiers: []string{"org"},
	Extensions:  []string{".org"},
	Parser:      OrgMolecule,
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
		res[""] = spaces.Fields(defs[0])
	}
	defs = defs[1:]

	for _, argspec := range defs {
		fields := spaces.Fields(argspec)
		res[fields[0]] = append(res[fields[0]], fields[1:]...)
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
