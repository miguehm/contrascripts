package parser

// Fountain is the root document node.
type Fountain struct {
	TitlePage TitlePage
	Elements  []Element
}

// TitlePage holds screenplay metadata.
type TitlePage struct {
	Title     string
	Credit    string
	Author    string
	DraftDate string
	Contact   string
	Source    string
	Revision  string
	Custom    map[string]string
}

// Element is the interface implemented by every AST node.
type Element interface {
	elementNode()
}

// SceneHeading represents a scene heading line (INT./EXT.).
type SceneHeading struct {
	Text string
}

// Action describes a line of action.
type Action struct {
	Text string
}

// Character identifies a speaking character.
type Character struct {
	Name      string
	Extension string
}

// Dialogue is the text spoken by a character.
type Dialogue struct {
	Text string
}

// Parenthetical is a direction within dialogue (whispered, beat, etc).
type Parenthetical struct {
	Text string
}

// Transition is a transition line (CUT TO:, FADE OUT., etc).
type Transition struct {
	Text string
}

// Centered is centered text enclosed in > <.
type Centered struct {
	Text string
}

// Note is a writer's note, invisible in output.
type Note struct {
	Text string
}

// Boneyard is a block comment /* ... */, invisible in output.
type Boneyard struct {
	Text string
}

// Section is a section heading (# Heading).
type Section struct {
	Level int
	Text  string
}

// Synopsis is a synopsis line (= Description).
type Synopsis struct {
	Text string
}

// Lyric is a musical lyric line (~ text).
type Lyric struct {
	Text string
}

// Forced is a line with a forced prefix (@, !, .).
type Forced struct {
	Prefix rune
	Text   string
}

// EmptyLine is a blank line used as a context separator.
type EmptyLine struct{}
