package parser

import (
	"strings"
	"testing"
)

func lintCodes(w []Warning) map[string]bool {
	m := map[string]bool{}
	for _, x := range w {
		m[x.Code] = true
	}
	return m
}

func TestLintCleanScript(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. HOUSE - DAY\n\nA man walks in.\n\nMAN\nHello there.\n"
	if w := Lint(input); len(w) != 0 {
		t.Errorf("expected no warnings, got %v", w)
	}
}

func TestLintEmpty(t *testing.T) {
	if w := Lint(""); len(w) != 0 {
		t.Errorf("expected no warnings, got %v", w)
	}
}

func TestLintUnclosedBoneyard(t *testing.T) {
	w := lintCodes(Lint("INT. X - DAY\n\n/*\nnever closed\n"))
	if !w["UNCLOSED_BONEYARD"] {
		t.Error("expected UNCLOSED_BONEYARD")
	}
}

func TestLintUnclosedNote(t *testing.T) {
	w := lintCodes(Lint("INT. X - DAY\n\n[[never closed\n"))
	if !w["UNCLOSED_NOTE"] {
		t.Error("expected UNCLOSED_NOTE")
	}
}

func TestLintUnclosedFrontmatter(t *testing.T) {
	w := lintCodes(Lint("---\ntitle: T\n"))
	if !w["UNCLOSED_FRONTMATTER"] {
		t.Error("expected UNCLOSED_FRONTMATTER")
	}
}

func TestLintUnclosedBold(t *testing.T) {
	w := lintCodes(Lint("INT. X - DAY\n\nAn **unclosed marker here.\n"))
	if !w["UNCLOSED_BOLD"] {
		t.Error("expected UNCLOSED_BOLD")
	}
}

func TestLintUnclosedItalic(t *testing.T) {
	w := lintCodes(Lint("INT. X - DAY\n\nAn _unclosed marker here.\n"))
	if !w["UNCLOSED_ITALIC"] {
		t.Error("expected UNCLOSED_ITALIC")
	}
}

func TestLintPossibleCharacter(t *testing.T) {
	input := "INT. X - DAY\nTHE DEALER eyes the player.\nSCANNING THE AISLES...\nWhere is he?\n"
	w := lintCodes(Lint(input))
	if !w["POSSIBLE_CHARACTER"] {
		t.Error("expected POSSIBLE_CHARACTER")
	}
}

func TestLintForcedActionNoWarning(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\n!SCANNING THE AISLES...\n\nMore action.\n"
	for _, x := range Lint(input) {
		if x.Code == "POSSIBLE_CHARACTER" {
			t.Errorf("unexpected POSSIBLE_CHARACTER: %v", x)
		}
	}
}

func TestLintOrphanDialogue(t *testing.T) {
	input := "INT. X - DAY\n\n(muttering)\nHello.\n"
	w := lintCodes(Lint(input))
	if !w["ORPHAN_PAREN"] {
		t.Error("expected ORPHAN_PAREN")
	}
}

func TestLintPossibleTransition(t *testing.T) {
	input := "INT. X - DAY\n\nCut to:\n\nEXT. Y - NIGHT\n"
	w := lintCodes(Lint(input))
	if !w["POSSIBLE_TRANSITION"] {
		t.Error("expected POSSIBLE_TRANSITION")
	}
	input2 := "INT. X - DAY\n\nSMASH CUT TO\n\nEXT. Y - NIGHT\n"
	if w2 := lintCodes(Lint(input2)); !w2["POSSIBLE_TRANSITION"] {
		t.Error("expected POSSIBLE_TRANSITION for missing colon")
	}
}

func TestLintPossibleScene(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT-CASA DIA\n\nAction.\n"
	w := lintCodes(Lint(input))
	if !w["POSSIBLE_SCENE"] {
		t.Error("expected POSSIBLE_SCENE")
	}
}

func TestLintNoFalseSceneOnWords(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nEstrellas cubren el vacio.\n\nInterior lights glow softly.\n"
	for _, x := range Lint(input) {
		if x.Code == "POSSIBLE_SCENE" {
			t.Errorf("unexpected POSSIBLE_SCENE: %v", x)
		}
	}
}

func TestLintBadSection(t *testing.T) {
	w := lintCodes(Lint("Title: T\nAuthor: A\n\nINT. X - DAY\n\n#nospace\n"))
	if !w["BAD_SECTION"] {
		t.Error("expected BAD_SECTION")
	}
}

func TestLintBadSynopsis(t *testing.T) {
	w := lintCodes(Lint("Title: T\nAuthor: A\n\nINT. X - DAY\n\n=nospace\n"))
	if !w["BAD_SYNOPSIS"] {
		t.Error("expected BAD_SYNOPSIS")
	}
}

func TestLintBadPagebreak(t *testing.T) {
	w := lintCodes(Lint("Title: T\nAuthor: A\n\nINT. X - DAY\n\n=== hello\n"))
	if !w["BAD_PAGEBREAK"] {
		t.Error("expected BAD_PAGEBREAK")
	}
}

func TestLintEmptyForced(t *testing.T) {
	w := lintCodes(Lint("Title: T\nAuthor: A\n\nINT. X - DAY\n\n@\n"))
	if !w["EMPTY_FORCED"] {
		t.Error("expected EMPTY_FORCED")
	}
}

func TestLintDialogBeforeScene(t *testing.T) {
	w := lintCodes(Lint("\nMAN\nHello.\n"))
	if !w["DIALOG_BEFORE_SCENE"] {
		t.Error("expected DIALOG_BEFORE_SCENE")
	}
}

func TestLintCharacterNoDialogue(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nMAN\n\nAction here.\n"
	w := lintCodes(Lint(input))
	if !w["CHARACTER_NO_DIALOGUE"] {
		t.Error("expected CHARACTER_NO_DIALOGUE")
	}
}

func TestLintDualMismatch(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nBRICK ^ STEEL\nDialogue.\n"
	w := lintCodes(Lint(input))
	if !w["DUAL_MISMATCH"] {
		t.Error("expected DUAL_MISMATCH")
	}
}

func TestLintDualPairClean(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nSTEEL\nScrew it.\n\nBRICK ^\nScrew it.\n"
	for _, x := range Lint(input) {
		if x.Code == "DUAL_MISMATCH" || x.Code == "NAME_INCONSISTENT" {
			t.Errorf("unexpected warning: %v", x)
		}
	}
}

func TestLintSceneEmpty(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nEXT. Y - NIGHT\n\nAction.\n"
	w := lintCodes(Lint(input))
	if !w["SCENE_EMPTY"] {
		t.Error("expected SCENE_EMPTY")
	}
}

func TestLintSceneNoTime(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. ABANDONED HOUSE\n\nAction.\n"
	w := lintCodes(Lint(input))
	if !w["SCENE_NO_TIME"] {
		t.Error("expected SCENE_NO_TIME")
	}
}

func TestLintForcedSceneNoTimeOk(t *testing.T) {
	input := "Title: T\nAuthor: A\n\n.SNIPER SCOPE POV\n\nAction.\n"
	for _, x := range Lint(input) {
		if x.Code == "SCENE_NO_TIME" {
			t.Errorf("unexpected SCENE_NO_TIME for forced heading: %v", x)
		}
	}
}

func TestLintTransitionChain(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nCUT TO:\n\nFADE TO:\n\nEXT. Y - NIGHT\n"
	w := lintCodes(Lint(input))
	if !w["TRANSITION_CHAIN"] {
		t.Error("expected TRANSITION_CHAIN")
	}
}

func TestLintBadExtension(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nMAN (this is a very long extension note here)\nHi.\n"
	w := lintCodes(Lint(input))
	if !w["BAD_EXTENSION"] {
		t.Error("expected BAD_EXTENSION")
	}
}

func TestLintTitleMissing(t *testing.T) {
	w := lintCodes(Lint("Contact: Studio\n\nINT. X - DAY\n"))
	if !w["TITLE_MISSING"] {
		t.Error("expected TITLE_MISSING")
	}
	if !w["AUTHOR_MISSING"] {
		t.Error("expected AUTHOR_MISSING")
	}
}

func TestLintBadRevisionColor(t *testing.T) {
	input := "Title: T\nAuthor: A\nRevision color: purple\n\nINT. X - DAY\n"
	w := lintCodes(Lint(input))
	if !w["BAD_REVISION_COLOR"] {
		t.Error("expected BAD_REVISION_COLOR")
	}
}

func TestLintBadSceneNumbers(t *testing.T) {
	input := "Title: T\nscene-numbers: maybe\n\nINT. X - DAY\n"
	w := lintCodes(Lint(input))
	if !w["BAD_SCENE_NUMBERS"] {
		t.Error("expected BAD_SCENE_NUMBERS")
	}
}

func TestLintLongAction(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nLine one.\nLine two.\nLine three.\nLine four.\nLine five.\n"
	w := lintCodes(Lint(input))
	if !w["LONG_ACTION"] {
		t.Error("expected LONG_ACTION")
	}
}

func TestLintLongDialogue(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nMAN\nOne.\nTwo.\nThree.\nFour.\nFive.\n"
	w := lintCodes(Lint(input))
	if !w["LONG_DIALOGUE"] {
		t.Error("expected LONG_DIALOGUE")
	}
}

func TestLintLongParen(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nMAN\n(starting the engine while looking back over his shoulder at the road)\nGo.\n"
	w := lintCodes(Lint(input))
	if !w["LONG_PAREN"] {
		t.Error("expected LONG_PAREN")
	}
}

func TestLintOveruseCentered(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\n>A<\n\n>B<\n\n>C<\n\n>D<\n\n>E<\n\n>F<\n"
	w := lintCodes(Lint(input))
	if !w["OVERUSE_CENTERED"] {
		t.Error("expected OVERUSE_CENTERED")
	}
}

func TestLintNameTypo(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nELENA\nHi.\n\nELANA\nHello.\n"
	w := lintCodes(Lint(input))
	if !w["NAME_INCONSISTENT"] {
		t.Error("expected NAME_INCONSISTENT")
	}
}

func TestLintNameExtensionClean(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nELENA\nHi.\n\nELENA (V.O.)\nHello.\n"
	for _, x := range Lint(input) {
		if x.Code == "NAME_INCONSISTENT" {
			t.Errorf("unexpected NAME_INCONSISTENT for extension: %v", x)
		}
	}
}

func TestLintStarItalicClean(t *testing.T) {
	input := "Title: T\nAuthor: A\n\nINT. X - DAY\n\nFace FILLS the *Leupold Mark 4* scope.\n"
	for _, x := range Lint(input) {
		if x.Code == "UNCLOSED_ITALIC" || x.Code == "UNCLOSED_BOLD" {
			t.Errorf("unexpected warning: %v", x)
		}
	}
}

func TestLintUnclosedStar(t *testing.T) {
	w := lintCodes(Lint("INT. X - DAY\n\nAn *unclosed marker here.\n"))
	if !w["UNCLOSED_ITALIC"] {
		t.Error("expected UNCLOSED_ITALIC")
	}
}

func TestLintSortedByLine(t *testing.T) {
	w := Lint("[[oops\n\nMAN\nHello.\n")
	for i := 1; i < len(w); i++ {
		if w[i].Line < w[i-1].Line {
			t.Fatalf("warnings not sorted: %v", w)
		}
	}
}

func TestLintWarningHasLine(t *testing.T) {
	w := Lint("INT. X - DAY\n\n/*\nnever closed\n")
	for _, x := range w {
		if x.Line < 1 || strings.TrimSpace(x.Code) == "" || strings.TrimSpace(x.Message) == "" {
			t.Errorf("malformed warning: %+v", x)
		}
	}
}
