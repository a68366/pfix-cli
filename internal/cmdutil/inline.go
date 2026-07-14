package cmdutil

import (
	"regexp"
	"strconv"
)

var uniqueIDRe = regexp.MustCompile(`uniqueid=(\d+)`)

// ScanFileIDs extracts Planfix inline-file ids from editor HTML. An inline
// upload is embedded as <img … src="…?action=getfile&uniqueid=<id>…">; the same
// id also appears in the element's class attribute, so matching only the
// uniqueid= token avoids counting each image twice. Ids are returned in order of
// first appearance, deduplicated.
func ScanFileIDs(html string) []int {
	matches := uniqueIDRe.FindAllStringSubmatch(html, -1)
	seen := make(map[int]bool, len(matches))
	var ids []int
	for _, m := range matches {
		id, err := strconv.Atoi(m[1])
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}
