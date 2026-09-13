// Package prompt resolves agent prompt templates by substituting {name}
// placeholders with values known at agent-construction time.
package prompt

import "strings"

// Resolve replaces each {key} in tmpl with vars[key]. Placeholders with no
// matching variable are left untouched, so unresolved ones are visible rather
// than silently dropped.
func Resolve(tmpl string, vars map[string]string) string {
	if tmpl == "" || len(vars) == 0 {
		return tmpl
	}
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(tmpl)
}
