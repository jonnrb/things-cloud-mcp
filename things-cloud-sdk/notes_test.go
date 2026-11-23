package thingscloud

import (
	"encoding/json"
	"testing"
)

func TestNote_FullText(t *testing.T) {
	raw := `{"_t":"tx","t":1,"ch":0,"v":"Hello world"}`
	var n Note
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatal(err)
	}
	if n.Type != NoteTypeFullText {
		t.Errorf("expected type %d, got %d", NoteTypeFullText, n.Type)
	}
	if n.Value != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", n.Value)
	}
}

func TestNote_Delta(t *testing.T) {
	raw := `{"_t":"tx","t":2,"ps":[{"r":"inserted text","p":0,"l":0,"ch":12345}]}`
	var n Note
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatal(err)
	}
	if n.Type != NoteTypeDelta {
		t.Errorf("expected type %d, got %d", NoteTypeDelta, n.Type)
	}
	if len(n.Patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(n.Patches))
	}
	if n.Patches[0].Replacement != "inserted text" {
		t.Errorf("unexpected replacement: %s", n.Patches[0].Replacement)
	}
}

func TestNote_ApplyPatch(t *testing.T) {
	original := "Hello world"
	patch := NotePatch{Position: 5, Length: 6, Replacement: " Go"}
	result := ApplyPatches(original, []NotePatch{patch})
	if result != "Hello Go" {
		t.Errorf("expected 'Hello Go', got '%s'", result)
	}
}

func TestNote_ApplyPatch_PositionBeyondLength(t *testing.T) {
	// Patch position is beyond the string — should not panic
	result := ApplyPatches("", []NotePatch{{Position: 10, Length: 0, Replacement: "hello"}})
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestNote_ApplyPatch_LengthBeyondEnd(t *testing.T) {
	result := ApplyPatches("AB", []NotePatch{{Position: 1, Length: 100, Replacement: "X"}})
	if result != "AX" {
		t.Errorf("expected 'AX', got '%s'", result)
	}
}

func TestNote_ApplyMultiplePatches(t *testing.T) {
	original := "ABCDEF"
	patches := []NotePatch{
		{Position: 0, Length: 1, Replacement: "X"},
	}
	result := ApplyPatches(original, patches)
	if result != "XBCDEF" {
		t.Errorf("expected 'XBCDEF', got '%s'", result)
	}
}

// nativeNote is the note of a task created in Things 3.23.3, and nativeDelta
// the patch the app wrote when "Second" was edited to "Updated" (from
// arthursoares/things-cloud-sdk's disposable-account check, 2026-09-06).
const nativeNote = "Native Task7 note α 🚀\nSecond line."

var nativeDelta = NotePatch{Replacement: "Update", Position: 26, Length: 5, Checksum: 3672733299}

func TestNoteChecksumMatchesThings(t *testing.T) {
	if got := NoteChecksum("Native Task7 note α 🚀\nUpdated line."); got != nativeDelta.Checksum {
		t.Fatalf("NoteChecksum = %d, want the native patch's %d", got, nativeDelta.Checksum)
	}
}

func TestApplyPatchesReadsByteOffsets(t *testing.T) {
	got, verified := ApplyPatchesVerified(nativeNote, []NotePatch{nativeDelta})
	if want := "Native Task7 note α 🚀\nUpdated line."; got != want {
		t.Fatalf("ApplyPatchesVerified = %q, want %q", got, want)
	}
	if !verified {
		t.Fatal("native patch not verified against its checksum")
	}
	if got := ApplyPatches(nativeNote, []NotePatch{nativeDelta}); got != "Native Task7 note α 🚀\nUpdated line." {
		t.Fatalf("ApplyPatches = %q", got)
	}
}

func TestApplyPatchesFallsBackToTheReadingTheChecksumNames(t *testing.T) {
	for _, tc := range []struct {
		name     string
		original string
		patch    NotePatch
		want     string
	}{
		{
			// UTF-16 offset 23 is "Second"; byte offset 23 splits the rocket,
			// and character offset 23 is its "e".
			name:     "UTF-16",
			original: nativeNote,
			patch:    NotePatch{Replacement: "Updated", Position: 23, Length: 6},
			want:     "Native Task7 note α 🚀\nUpdated line.",
		},
		{
			// Character offset 2 is "b"; UTF-16 offset 2 is the space, and byte
			// offset 2 splits the rocket.
			name:     "characters",
			original: "🚀 b",
			patch:    NotePatch{Replacement: "c", Position: 2, Length: 1},
			want:     "🚀 c",
		},
		{
			// Byte offset 3 is "b"; at character and UTF-16 offsets, 3 is the
			// end of the note.
			name:     "bytes",
			original: "é b",
			patch:    NotePatch{Replacement: "c", Position: 3, Length: 1},
			want:     "é c",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.patch.Checksum = NoteChecksum(tc.want)
			got, verified := ApplyPatchesVerified(tc.original, []NotePatch{tc.patch})
			if got != tc.want || !verified {
				t.Fatalf("ApplyPatchesVerified = %q, %v; want %q, true", got, verified, tc.want)
			}
		})
	}
}

func TestApplyPatchesChecksEachPatch(t *testing.T) {
	first := NotePatch{Replacement: "Update", Position: 26, Length: 5}
	first.Checksum = NoteChecksum("Native Task7 note α 🚀\nUpdated line.")
	second := NotePatch{Replacement: "β", Position: 18, Length: 2}
	second.Checksum = NoteChecksum("Native Task7 note β 🚀\nUpdated line.")
	got, verified := ApplyPatchesVerified(nativeNote, []NotePatch{first, second})
	if want := "Native Task7 note β 🚀\nUpdated line."; got != want || !verified {
		t.Fatalf("ApplyPatchesVerified = %q, %v; want %q, true", got, verified, want)
	}
}

func TestApplyPatchesWithoutAChecksumReadsBytes(t *testing.T) {
	got, verified := ApplyPatchesVerified(nativeNote, []NotePatch{{Replacement: "Update", Position: 26, Length: 5}})
	if want := "Native Task7 note α 🚀\nUpdated line."; got != want || !verified {
		t.Fatalf("ApplyPatchesVerified = %q, %v; want %q, true", got, verified, want)
	}
}

func TestApplyPatchesNeverSplitsACharacter(t *testing.T) {
	// Byte offset 1 is inside "é", so with no checksum to go by, the patch is
	// read at character offsets.
	got, _ := ApplyPatchesVerified("éa", []NotePatch{{Replacement: "b", Position: 1, Length: 1}})
	if got != "éb" {
		t.Fatalf("ApplyPatchesVerified = %q, want %q", got, "éb")
	}
	got, verified := ApplyPatchesVerified("éa", []NotePatch{{Replacement: "b", Position: 1, Length: 1, Checksum: 1}})
	if got != "éb" || verified {
		t.Fatalf("ApplyPatchesVerified with an unmatched checksum = %q, %v; want %q, false", got, verified, "éb")
	}
}

func TestApplyPatchesReportsAnUnmatchedChecksum(t *testing.T) {
	patch := nativeDelta
	patch.Checksum = 1
	got, verified := ApplyPatchesVerified(nativeNote, []NotePatch{patch})
	if want := "Native Task7 note α 🚀\nUpdated line."; got != want {
		t.Fatalf("ApplyPatchesVerified = %q, want the byte reading %q", got, want)
	}
	if verified {
		t.Fatal("a checksum no reading produces was reported verified")
	}
}
