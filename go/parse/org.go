package parse

import (
	"fmt"
	"strings"
)

///////////////////
// Text matching //
///////////////////

var orgSectionRe = re(`^(\*+) (.+)$`)
var orgBeginSrcPfx = str("#+begin_src")
var orgEndSrcPfx = str("#+end_src")
var orgPropertyPfx = str("#+")
var orgBeginPfx = str("#+begin_")
var orgEndPfx = str("#+end_")

////////////
// Makers //
////////////

// OrgCodeMk makes a code element from Org lines.
func OrgCodeMk(lines []string) ElementImpl {
	lang, params := ParseOrgBeginSrc(lines[0])
	return CodeElement{
		Raw:    lines[1 : len(lines)-1],
		Lang:   lang,
		Params: params,
	}
}

// OrgBlockMk makes a block element from Org lines.
func OrgBlockMk(lines []string) ElementImpl {
	return BlockElement{
		Raw:  lines[1 : len(lines)-1],
		Type: orgBeginPfx.StripLeftOf(lines[0]),
	}
}

// OrgPropertyMk makes a metadata element from an Org property line.
func OrgPropertyMk(lines []string) ElementImpl {
	line := lines[0]
	split := strings.SplitN(line, ":", 2)
	res := MetadataElement{Name: spaces.Trim(split[0])}
	if len(split) == 2 {
		res.RawValue = spaces.TrimRight(split[1])
	}
	return res
}

///////////////////////////////////
// High-level parsing and fusing //
///////////////////////////////////

// OrgRules is a sequence of rules able to parse an Org file.
var OrgRules = Rules{
	Rule{ // Section, hierarchical delimiter of the document.
		Take: FirstTake(orgSectionRe.Match),
		Bake: NoBk,
		Make: ReSectionMake(orgSectionRe),
	},
	Rule{ // Code, content meant for machine consumption.
		Take: BetweenTake(orgBeginSrcPfx.IsPrefix, orgEndSrcPfx.IsPrefix),
		Bake: NoBk,
		Make: OrgCodeMk,
	},
	Rule{ // Other kind of blocks, like quote blocks.
		// This taker doesn't ensure that the begin and end block are matching.
		// It will work fine assuming no wild ^#+end_ is present inside blocks.
		// This is bound to happen eventually so I guess this is a TODO.
		Take: BetweenTake(orgBeginPfx.IsPrefix, orgEndPfx.IsPrefix),
		Bake: NoBk,
		Make: OrgBlockMk,
	},
	Rule{ // Metadata about the document.
		Take: FirstTake(orgPropertyPfx.IsPrefix),
		Bake: orgPropertyPfx.StripLeftOf,
		Make: OrgPropertyMk,
	},
	SpaceRule, // Whitespace, content that can typically be ignored.
	Rule{ // Prose, content meant for human consumption.
		Take: TrailingTake(spaces.Intersects, nor(orgSectionRe.Match, orgPropertyPfx.IsPrefix)),
		Bake: NoBk,
		Make: ProseMk,
	},
}

// OrgFuser can reconstruct the lines of an Org document from parsed elements.
func OrgFuser(matter Elements) ([]string, error) {
	res := slice[string]{}
	for _, part := range matter {
		switch p := part.ElementImpl.(type) {
		case CodeElement:
			begin := string(orgBeginSrcPfx) + " " + p.Lang
			if len(p.Params) > 0 {
				begin += " " + p.Params.FuseToNoweb()
			}
			res.Add(begin)
			res.Add(p.Raw...)
			res.Add(string(orgEndSrcPfx))

		case ProseElement:
			res.Add(p.Raw...)

		case MetadataElement:
			prop := "#+" + p.Name + ":"
			if p.RawValue != "" {
				prop += p.RawValue
			}
			res.Add(prop)

		case SectionElement:
			res.Add(strings.Repeat("*", p.Level) + " " + p.Title)

		case SpaceElement:
			res.Add(p.Raw...)

		case BlockElement:
			res.Add(string(orgBeginPfx) + p.Type)
			res.Add(p.Raw...)
			res.Add(string(orgEndPfx) + p.Type)

		default:
			return nil, fmt.Errorf("no org fuser for %T", part.ElementImpl)
		}
	}
	return res, nil
}

// OrgLang holds information needed to manipulate Org files.
var OrgLang = Language{
	Identifiers: []string{"org"},
	Extensions:  []string{".org"},
	Parser:      OrgRules,
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
