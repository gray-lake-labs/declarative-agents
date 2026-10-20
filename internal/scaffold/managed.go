package scaffold

import (
	"bytes"
	"strings"
)

// BlockBegin and BlockEnd bound tpm's text in an orientation file. A later
// tpm replaces what lies between them and keeps every line outside them
// (srd043 R3.5).
const (
	BlockBegin = "<!-- tpm:begin: tpm writes the text down to tpm:end, and tpm doctor refreshes it; write your own notes outside it -->"
	BlockEnd   = "<!-- tpm:end -->"
)

// blockBeginPrefix matches a begin line whatever note it carries, so a
// reworded note still opens the block.
const blockBeginPrefix = "<!-- tpm:begin"

// splitManaged cuts content around its managed block: the lines before the
// begin line, the block from the begin line through the end line, and the
// lines after it. ok is false when content holds no whole block.
func splitManaged(content []byte) (before, block, after []byte, ok bool) {
	lines := bytes.SplitAfter(content, []byte("\n"))
	begin, end := -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(string(l))
		switch {
		case begin < 0 && strings.HasPrefix(t, blockBeginPrefix):
			begin = i
		case begin >= 0 && t == BlockEnd:
			end = i
		}
		if end >= 0 {
			break
		}
	}
	if begin < 0 || end < 0 {
		return nil, nil, nil, false
	}
	return bytes.Join(lines[:begin], nil), bytes.Join(lines[begin:end+1], nil), bytes.Join(lines[end+1:], nil), true
}

// refreshManaged is cur with its managed block replaced by fresh's. ok is
// false when either has no whole block, so the file is not tpm's to refresh.
func refreshManaged(cur, fresh []byte) ([]byte, bool) {
	before, _, after, ok := splitManaged(cur)
	if !ok {
		return nil, false
	}
	_, block, _, ok := splitManaged(fresh)
	if !ok {
		return nil, false
	}
	return append(append(append([]byte{}, before...), block...), after...), true
}
