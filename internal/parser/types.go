package parser

// Fountain is the root document node.
type Fountain struct {
	TitlePage TitlePage
	Elements  []Element
}

// TitlePage holds screenplay metadata. The json tags define the wire format
// consumed by non-Go renderers such as the web preview.
type TitlePage struct {
	Title         string            `json:"title"`
	Credit        string            `json:"credit"`
	Author        string            `json:"author"`
	DraftDate     string            `json:"draftDate"`
	Contact       string            `json:"contact"`
	Source        string            `json:"source"`
	Revision      string            `json:"revision"`
	Logline       string            `json:"logline"`
	Tagline       string            `json:"tagline"`
	Synopsis      string            `json:"synopsis"`
	Genre         string            `json:"genre"`
	Subgenre      string            `json:"subgenre"`
	Tone          string            `json:"tone"`
	Themes        string            `json:"themes"`
	Comps         string            `json:"comps"`
	Format        string            `json:"format"`
	Season        string            `json:"season"`
	Episode       string            `json:"episode"`
	EpisodeTitle  string            `json:"episodeTitle"`
	Runtime       string            `json:"runtime"`
	Pages         string            `json:"pages"`
	Language      string            `json:"language"`
	Status        string            `json:"status"`
	ProdCompany   string            `json:"prodCompany"`
	Producer      string            `json:"producer"`
	Director      string            `json:"director"`
	Budget        string            `json:"budget"`
	AdaptedBy     string            `json:"adaptedBy"`
	BasedOn       string            `json:"basedOn"`
	Copyright     string            `json:"copyright"`
	WGA           string            `json:"wga"`
	Rights        string            `json:"rights"`
	Dedication    string            `json:"dedication"`
	Disclaimer    string            `json:"disclaimer"`
	DraftType     string            `json:"draftType"`
	Version       string            `json:"version"`
	RevisionColor string            `json:"revisionColor"`
	RevisionDate  string            `json:"revisionDate"`
	Email         string            `json:"email"`
	Phone         string            `json:"phone"`
	Agent         string            `json:"agent"`
	Manager       string            `json:"manager"`
	Address       string            `json:"address"`
	SceneNumbers  bool              `json:"sceneNumbers"`
	Custom        map[string]string `json:"custom"`
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

// Character identifies a speaking character, stored literally as written.
type Character struct {
	Text string
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
