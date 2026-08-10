package skills

import "embed"

// Files embeds the harbor skill — the single skill shipped with the binary.
// It teaches the leading word `harbor` and the whole page workflow: create the
// workspace first, create tags first, import pages with provenance, search
// before updating, respond to the comments queue. One install delivers the
// workflow for producing, organizing, finding, and reviewing pages.
//
//go:embed harbor
var Files embed.FS

// All is the list of embedded skills installed by `harbor skills install`.
var All = []string{"harbor"}

// SkillName is the primary (and only) shipped skill.
const SkillName = "harbor"
