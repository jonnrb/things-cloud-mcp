package thingscloud

import (
	"hash/crc32"
	"unicode/utf16"
	"unicode/utf8"
)

// NoteTypeFullText indicates a note with complete text
const NoteTypeFullText = 1

// NoteTypeDelta indicates a note with incremental patches
const NoteTypeDelta = 2

// NotePatch describes a single text replacement operation
type NotePatch struct {
	Replacement string `json:"r"`
	Position    int    `json:"p"`
	Length      int    `json:"l"`
	Checksum    int64  `json:"ch"`
}

// Note describes a structured note as used by the Things API
type Note struct {
	TypeTag  string      `json:"_t"`
	Type     int         `json:"t"`
	Checksum int64       `json:"ch,omitempty"`
	Value    string      `json:"v,omitempty"`
	Patches  []NotePatch `json:"ps,omitempty"`
}

// NoteChecksum is the checksum Things keeps with a note: the CRC-32 (IEEE) of
// its UTF-8 text. A full-text note carries the checksum of its text, and each
// patch of a delta carries the checksum of the note after that patch.
func NoteChecksum(text string) int64 {
	return int64(crc32.ChecksumIEEE([]byte(text)))
}

// ApplyPatches applies a note delta's patches to original, in order. See
// ApplyPatchesVerified.
func ApplyPatches(original string, patches []NotePatch) string {
	text, _ := ApplyPatchesVerified(original, patches)
	return text
}

// ApplyPatchesVerified applies a note delta's patches to original, in order,
// and reports whether each patch that carries a checksum produced the note it
// names.
//
// Things 3.23 and later address a patch by UTF-8 byte offsets: after "α 🚀\n",
// the next character is at 8, not 4 as counted in characters. Reading those
// offsets as characters, as this package used to, puts every edit after a
// curly quote, accent or emoji in the wrong place. A patch is therefore read
// at byte offsets first. Where that doesn't produce the patch's checksum, it is
// read again at UTF-16 and at character offsets, which other versions of
// Things may have used, and the reading that does produce it wins. A patch with
// no checksum, or one no reading matches, keeps the byte reading, unless that
// splits a character, when the character reading is kept.
func ApplyPatchesVerified(original string, patches []NotePatch) (string, bool) {
	text := original
	verified := true
	for _, p := range patches {
		var ok bool
		text, ok = applyPatch(text, p)
		verified = verified && ok
	}
	return text, verified
}

// applyPatch applies one patch, as ApplyPatchesVerified describes, and reports
// whether it has no checksum or a reading produced it.
func applyPatch(text string, p NotePatch) (string, bool) {
	byteReading, byteOK := patchBytes(text, p)
	if p.Checksum == 0 {
		if byteOK {
			return byteReading, true
		}
		return patchRunes(text, p), true
	}
	if byteOK && NoteChecksum(byteReading) == p.Checksum {
		return byteReading, true
	}
	if utf16Reading, ok := patchUTF16(text, p); ok && NoteChecksum(utf16Reading) == p.Checksum {
		return utf16Reading, true
	}
	runeReading := patchRunes(text, p)
	if NoteChecksum(runeReading) == p.Checksum {
		return runeReading, true
	}
	if byteOK {
		return byteReading, false
	}
	return runeReading, false
}

// span clamps a patch's position and length to a text of n units and returns
// the start and end of the range it replaces.
func span(p NotePatch, n int) (int, int) {
	start := p.Position
	if start < 0 {
		start = 0
	}
	if start > n {
		start = n
	}
	length := p.Length
	if length < 0 {
		length = 0
	}
	if length > n-start {
		length = n - start
	}
	return start, start + length
}

// patchBytes reads the patch at UTF-8 byte offsets. It reports false if the
// range would split a character.
func patchBytes(text string, p NotePatch) (string, bool) {
	start, end := span(p, len(text))
	if !isCharBoundary(text, start) || !isCharBoundary(text, end) {
		return "", false
	}
	return text[:start] + p.Replacement + text[end:], true
}

func isCharBoundary(text string, i int) bool {
	return i == 0 || i == len(text) || utf8.RuneStart(text[i])
}

// patchUTF16 reads the patch at UTF-16 code unit offsets. It reports false if
// the range would split a surrogate pair.
func patchUTF16(text string, p NotePatch) (string, bool) {
	units := utf16.Encode([]rune(text))
	start, end := span(p, len(units))
	if splitsPair(units, start) || splitsPair(units, end) {
		return "", false
	}
	return string(utf16.Decode(units[:start])) + p.Replacement + string(utf16.Decode(units[end:])), true
}

func splitsPair(units []uint16, i int) bool {
	if i <= 0 || i >= len(units) {
		return false
	}
	high, low := units[i-1], units[i]
	return high >= 0xd800 && high < 0xdc00 && low >= 0xdc00 && low < 0xe000
}

// patchRunes reads the patch at character (Unicode code point) offsets.
func patchRunes(text string, p NotePatch) string {
	runes := []rune(text)
	start, end := span(p, len(runes))
	return string(runes[:start]) + p.Replacement + string(runes[end:])
}
