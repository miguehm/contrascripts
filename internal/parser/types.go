package parser

// Fountain is the root document node.
type Fountain struct {
	TitlePage TitlePage
	Elements  []Element
}

// TitlePage holds screenplay metadata.
type TitlePage struct {
	Title         string
	Credit        string
	Author        string
	DraftDate     string
	Contact       string
	Source        string
	Revision      string
	Logline       string
	Tagline       string
	Synopsis      string
	Genre         string
	Subgenre      string
	Tone          string
	Themes        string
	Comps         string
	Format        string
	Season        string
	Episode       string
	EpisodeTitle  string
	Runtime       string
	Pages         string
	Language      string
	Status        string
	ProdCompany   string
	Producer      string
	Director      string
	Budget        string
	AdaptedBy     string
	BasedOn       string
	Copyright     string
	WGA           string
	Rights        string
	Dedication    string
	Disclaimer    string
	DraftType     string
	Version       string
	RevisionColor string
	RevisionDate  string
	Email         string
	Phone         string
	Agent         string
	Manager       string
	Address       string
	Custom        map[string]string
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

// PageBreak represents a forced page break (===).
type PageBreak struct{}

// InlineElement is the interface implemented by every inline formatting node.
type InlineElement interface {
	inlineNode()
}

// PlainText represents unformatted plain text within an inline element.
type PlainText struct {
	Text string
}

// Bold represents bold text enclosed in ** **.
type Bold struct {
	Content []InlineElement
}

// Italic represents italic text enclosed in _ _.
type Italic struct {
	Content []InlineElement
}

// --- Element interface implementations ---

func (SceneHeading) elementNode()  {}
func (Action) elementNode()        {}
func (Character) elementNode()     {}
func (Dialogue) elementNode()      {}
func (Parenthetical) elementNode() {}
func (Transition) elementNode()    {}
func (Centered) elementNode()      {}
func (Note) elementNode()          {}
func (Boneyard) elementNode()      {}
func (Section) elementNode()       {}
func (Synopsis) elementNode()      {}
func (Lyric) elementNode()         {}
func (Forced) elementNode()        {}
func (EmptyLine) elementNode()     {}
func (PageBreak) elementNode()     {}

// --- InlineElement interface implementations ---

func (PlainText) inlineNode() {}
func (Bold) inlineNode()      {}
func (Italic) inlineNode()    {}
