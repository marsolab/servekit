package slackkit

import (
	"fmt"
)

const (
	maxMessageBlocks   = 50
	maxSectionFields   = 10
	maxContextElements = 10
	maxActionElements  = 25
)

// Error is a custom error type for Slack notifications.
type Error string

func (e Error) Error() string { return string(e) }

// Slack notification errors.
const (
	// ErrNilBlock is returned when a block is nil.
	ErrNilBlock Error = "nil block"

	// ErrNilText is returned when a text is nil.
	ErrNilText Error = "nil text"

	// ErrInvalidBlockType is returned when a block type is invalid.
	ErrInvalidBlockType Error = "invalid block type"

	// ErrInvalidBlockTextType is returned when a block text type is invalid.
	ErrInvalidBlockTextType Error = "invalid block text type"

	// ErrInvalidBlockTextStyle is returned when a block text style is invalid.
	ErrInvalidBlockTextStyle Error = "invalid block text style"

	// ErrInvalidBlockTextEmoji is returned when a block text emoji is invalid.
	ErrInvalidBlockTextEmoji Error = "invalid block text emoji"

	// ErrInvalidBlockTextVerbatim is returned when a text verbatim value is invalid.
	ErrInvalidBlockTextVerbatim Error = "invalid block text verbatim"

	// ErrTooManyBlocks is returned when a message contains too many blocks.
	ErrTooManyBlocks Error = "too many blocks"

	// ErrNilBlockFields is returned when block fields are empty.
	ErrNilBlockFields Error = "nil block fields"

	// ErrTooManyBlockFields is returned when a section contains too many fields.
	ErrTooManyBlockFields Error = "too many block fields"

	// ErrNilBlockElements is returned when block elements are empty.
	ErrNilBlockElements Error = "nil block elements"

	// ErrTooManyBlockElements is returned when a block contains too many elements.
	ErrTooManyBlockElements Error = "too many block elements"

	// ErrInvalidElementType is returned when a block element type is invalid.
	ErrInvalidElementType Error = "invalid element type"

	// ErrInvalidButtonStyle is returned when a button style is invalid.
	ErrInvalidButtonStyle Error = "invalid button style"

	// ErrNilImageURL is returned when an image URL is empty.
	ErrNilImageURL Error = "nil image url"

	// ErrNilAltText is returned when image alt text is empty.
	ErrNilAltText Error = "nil alt text"

	// ErrNilChannel is returned when a Slack channel is empty.
	ErrNilChannel Error = "nil channel"
)

// NewNotification creates a new notification.
func NewNotification(blocks ...Block) (*Notification, error) {
	notification := Notification{Blocks: blocks}
	if err := notification.validate(); err != nil {
		return nil, err
	}

	return &notification, nil
}

// NewMessage creates a new notification with fallback text for Slack notifications.
func NewMessage(text string, blocks ...Block) (*Notification, error) {
	if text == "" {
		return nil, ErrNilText
	}

	notification := Notification{
		Text:   text,
		Blocks: blocks,
	}
	if err := notification.validate(); err != nil {
		return nil, err
	}

	return &notification, nil
}

// BlockType values are defined in the Slack API documentation.
type BlockType string

// Block types are defined in the Slack API documentation.
const (
	Header     BlockType = "header"
	Divider    BlockType = "divider"
	Section    BlockType = "section"
	Context    BlockType = "context"
	Actions    BlockType = "actions"
	ImageBlock BlockType = "image"
)

// TextType values are defined in the Slack API documentation.
type TextType string

const (
	// PlainText is the default text type.
	PlainText TextType = "plain_text"

	// Markdown is the markdown text type.
	Markdown TextType = "mrkdwn"
)

// ElementType values are defined in the Slack API documentation.
type ElementType string

// Element types are defined in the Slack API documentation.
const (
	ButtonElement    ElementType = "button"
	ImageElementType ElementType = "image"
)

// ButtonStyle values are defined in the Slack API documentation.
type ButtonStyle string

// Button styles are defined in the Slack API documentation.
const (
	PrimaryButton ButtonStyle = "primary"
	DangerButton  ButtonStyle = "danger"
)

// Notification is the main struct for sending notifications to Slack.
//
//nolint:tagliatelle // Slack API uses snake_case JSON fields.
type Notification struct {
	Text     string  `json:"text,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
	ThreadTS string  `json:"thread_ts,omitempty"`
}

// Block is a single block of a notification.
//
//nolint:tagliatelle // Slack API uses snake_case JSON fields.
type Block struct {
	Type      BlockType `json:"type"`
	BlockID   string    `json:"block_id,omitempty"`
	Text      *Text     `json:"text,omitempty"`
	Fields    []Text    `json:"fields,omitempty"`
	Accessory any       `json:"accessory,omitempty"`
	Elements  []any     `json:"elements,omitempty"`
	ImageURL  string    `json:"image_url,omitempty"`
	AltText   string    `json:"alt_text,omitempty"`
	Title     *Text     `json:"title,omitempty"`
	Expand    *bool     `json:"expand,omitempty"`
}

// Text is a single text block of a notification.
type Text struct {
	Type     TextType `json:"type"`
	Text     string   `json:"text"`
	Emoji    *bool    `json:"emoji,omitempty"`
	Verbatim *bool    `json:"verbatim,omitempty"`
	Style    *Style   `json:"style,omitempty"`
}

// Style is a style block of a notification.
type Style struct {
	Bold   *bool `json:"bold,omitempty"`
	Italic *bool `json:"italic,omitempty"`
	Strike *bool `json:"strike,omitempty"`
}

// ImageElement is an image element used as a section accessory or context element.
//
//nolint:tagliatelle // Slack API uses snake_case JSON fields.
type ImageElement struct {
	Type     ElementType `json:"type"`
	ImageURL string      `json:"image_url"`
	AltText  string      `json:"alt_text"`
}

// Button is a Slack button element.
//
//nolint:tagliatelle // Slack API uses snake_case JSON fields.
type Button struct {
	Type               ElementType `json:"type"`
	Text               *Text       `json:"text"`
	ActionID           string      `json:"action_id,omitempty"`
	URL                string      `json:"url,omitempty"`
	Value              string      `json:"value,omitempty"`
	Style              ButtonStyle `json:"style,omitempty"`
	AccessibilityLabel string      `json:"accessibility_label,omitempty"`
}

// ContextText is a Slack text object used in context blocks.
type ContextText Text

// SectionAccessory is an element that can be attached to a section block.
type SectionAccessory interface {
	sectionAccessory()
}

// ContextElement is an element that can be used in a context block.
type ContextElement interface {
	contextElement()
}

// ActionElement is an element that can be used in an actions block.
type ActionElement interface {
	actionElement()
}

// BlockOption configures a Slack block.
type BlockOption func(*Block)

// WithBlockID sets the block identifier.
func WithBlockID(blockID string) BlockOption {
	return func(block *Block) {
		block.BlockID = blockID
	}
}

// WithAccessory sets a section accessory.
func WithAccessory(accessory SectionAccessory) BlockOption {
	return func(block *Block) {
		block.Accessory = accessory
	}
}

// WithImageTitle sets an image block title.
func WithImageTitle(title string, emoji bool) BlockOption {
	return func(block *Block) {
		block.Title = textPtr(NewPlainText(title, emoji))
	}
}

// WithExpand sets whether Slack should always expand section text.
func WithExpand(expand bool) BlockOption {
	return func(block *Block) {
		block.Expand = boolPtr(expand)
	}
}

// TextOption configures a Slack text object.
type TextOption func(*Text)

// WithVerbatim controls Slack automatic parsing for markdown text objects.
func WithVerbatim(verbatim bool) TextOption {
	return func(text *Text) {
		text.Verbatim = boolPtr(verbatim)
	}
}

// ButtonOption configures a Slack button.
type ButtonOption func(*Button)

// WithButtonStyle sets a button style.
func WithButtonStyle(style ButtonStyle) ButtonOption {
	return func(button *Button) {
		button.Style = style
	}
}

// WithButtonURL sets a button URL.
func WithButtonURL(url string) ButtonOption {
	return func(button *Button) {
		button.URL = url
	}
}

// WithButtonValue sets a button value.
func WithButtonValue(value string) ButtonOption {
	return func(button *Button) {
		button.Value = value
	}
}

// WithButtonAccessibilityLabel sets a button accessibility label.
func WithButtonAccessibilityLabel(label string) ButtonOption {
	return func(button *Button) {
		button.AccessibilityLabel = label
	}
}

// NewPlainText creates a plain text object.
func NewPlainText(text string, emoji bool) Text {
	return Text{
		Type:  PlainText,
		Text:  text,
		Emoji: boolPtr(emoji),
	}
}

// NewMarkdownText creates a markdown text object.
func NewMarkdownText(text string, options ...TextOption) Text {
	markdown := Text{
		Type: Markdown,
		Text: text,
	}

	for _, option := range options {
		option(&markdown)
	}

	return markdown
}

// NewHeader creates a new header block.
func NewHeader(text string, emoji bool, options ...BlockOption) Block {
	block := Block{
		Type: Header,
		Text: textPtr(NewPlainText(text, emoji)),
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewDivider creates a new divider block.
func NewDivider(options ...BlockOption) Block {
	block := Block{
		Type: Divider,
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewSection creates a new plain text section block.
func NewSection(text string, emoji bool, options ...BlockOption) Block {
	block := Block{
		Type: Section,
		Text: textPtr(NewPlainText(text, emoji)),
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewMarkdownSection creates a new markdown section block.
func NewMarkdownSection(text string, options ...BlockOption) Block {
	block := Block{
		Type: Section,
		Text: textPtr(NewMarkdownText(text)),
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewSectionFields creates a new section block with compact fields.
func NewSectionFields(fields []Text, options ...BlockOption) Block {
	block := Block{
		Type:   Section,
		Fields: fields,
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewImageBlock creates a new image block.
func NewImageBlock(imageURL, altText string, options ...BlockOption) Block {
	block := Block{
		Type:     ImageBlock,
		ImageURL: imageURL,
		AltText:  altText,
	}

	for _, option := range options {
		option(&block)
	}

	return block
}

// NewImageElement creates an image element for section accessories or context blocks.
func NewImageElement(imageURL, altText string) ImageElement {
	return ImageElement{
		Type:     ImageElementType,
		ImageURL: imageURL,
		AltText:  altText,
	}
}

// NewContextImage creates an image element for a context block.
func NewContextImage(imageURL, altText string) ImageElement {
	return NewImageElement(imageURL, altText)
}

// NewContextText creates a text element for a context block.
func NewContextText(text Text) ContextText {
	return ContextText(text)
}

// NewContext creates a context block.
func NewContext(elements ...ContextElement) Block {
	block := Block{
		Type:     Context,
		Elements: make([]any, 0, len(elements)),
	}

	for _, element := range elements {
		block.Elements = append(block.Elements, element)
	}

	return block
}

// NewActions creates an actions block.
func NewActions(elements ...ActionElement) Block {
	block := Block{
		Type:     Actions,
		Elements: make([]any, 0, len(elements)),
	}

	for _, element := range elements {
		block.Elements = append(block.Elements, element)
	}

	return block
}

// NewButton creates a button element.
func NewButton(text, actionID string, options ...ButtonOption) Button {
	button := Button{
		Type:     ButtonElement,
		Text:     textPtr(NewPlainText(text, true)),
		ActionID: actionID,
	}

	for _, option := range options {
		option(&button)
	}

	return button
}

func (n *Notification) validate() error {
	if n == nil {
		return ErrNilText
	}

	if len(n.Blocks) > maxMessageBlocks {
		return ErrTooManyBlocks
	}

	for i := range n.Blocks {
		err := n.Blocks[i].validate()
		if err != nil {
			return fmt.Errorf("block %d is invalid: %w", i, err)
		}
	}

	return nil
}

// validate checks if the block is valid.
func (b *Block) validate() error {
	if b == nil {
		return ErrNilBlock
	}

	switch b.Type {
	case Header:
		return b.validateHeader()

	case Divider:
		return nil

	case Section:
		return b.validateSection()

	case Context:
		return b.validateContext()

	case Actions:
		return b.validateActions()

	case ImageBlock:
		return b.validateImageBlock()

	default:
		return ErrInvalidBlockType
	}
}

func (b *Block) validateSection() error {
	if b.Text == nil && len(b.Fields) == 0 {
		return ErrNilText
	}

	if b.Text != nil {
		if err := b.Text.validate(PlainText, Markdown); err != nil {
			return err
		}
	}

	if len(b.Fields) > maxSectionFields {
		return ErrTooManyBlockFields
	}

	for i := range b.Fields {
		if err := b.Fields[i].validate(PlainText, Markdown); err != nil {
			return fmt.Errorf("field %d is invalid: %w", i, err)
		}
	}

	if b.Accessory != nil {
		if err := validateSectionAccessory(b.Accessory); err != nil {
			return err
		}
	}

	return nil
}

func (b *Block) validateHeader() error {
	if b.Text == nil {
		return ErrNilText
	}

	return b.Text.validate(PlainText)
}

func (b *Block) validateContext() error {
	if len(b.Elements) == 0 {
		return ErrNilBlockElements
	}

	if len(b.Elements) > maxContextElements {
		return ErrTooManyBlockElements
	}

	for i, element := range b.Elements {
		err := validateContextElement(element)
		if err != nil {
			return fmt.Errorf("element %d is invalid: %w", i, err)
		}
	}

	return nil
}

func (b *Block) validateActions() error {
	if len(b.Elements) == 0 {
		return ErrNilBlockElements
	}

	if len(b.Elements) > maxActionElements {
		return ErrTooManyBlockElements
	}

	for i, element := range b.Elements {
		err := validateActionElement(element)
		if err != nil {
			return fmt.Errorf("element %d is invalid: %w", i, err)
		}
	}

	return nil
}

func (b *Block) validateImageBlock() error {
	if b.ImageURL == "" {
		return ErrNilImageURL
	}

	if b.AltText == "" {
		return ErrNilAltText
	}

	if b.Title != nil {
		if err := b.Title.validate(PlainText); err != nil {
			return err
		}
	}

	return nil
}

func (t *Text) validate(allowedTypes ...TextType) error {
	if t == nil {
		return ErrNilText
	}

	if t.Style != nil {
		return ErrInvalidBlockTextStyle
	}

	if !containsTextType(allowedTypes, t.Type) {
		return ErrInvalidBlockTextType
	}

	if t.Type == Markdown && t.Emoji != nil {
		return ErrInvalidBlockTextEmoji
	}

	if t.Type == PlainText && t.Verbatim != nil {
		return ErrInvalidBlockTextVerbatim
	}

	if t.Text == "" {
		return ErrNilText
	}

	return nil
}

func validateSectionAccessory(accessory any) error {
	switch value := accessory.(type) {
	case ImageElement:
		return value.validate()
	case *ImageElement:
		if value == nil {
			return ErrInvalidElementType
		}

		return value.validate()
	case Button:
		return value.validate()
	case *Button:
		if value == nil {
			return ErrInvalidElementType
		}

		return value.validate()
	default:
		return ErrInvalidElementType
	}
}

func validateContextElement(element any) error {
	switch value := element.(type) {
	case ContextText:
		text := Text(value)

		return text.validate(PlainText, Markdown)
	case *ContextText:
		if value == nil {
			return ErrInvalidElementType
		}

		text := Text(*value)

		return text.validate(PlainText, Markdown)
	case ImageElement:
		return value.validate()
	case *ImageElement:
		if value == nil {
			return ErrInvalidElementType
		}

		return value.validate()
	default:
		return ErrInvalidElementType
	}
}

func validateActionElement(element any) error {
	switch value := element.(type) {
	case Button:
		return value.validate()
	case *Button:
		if value == nil {
			return ErrInvalidElementType
		}

		return value.validate()
	default:
		return ErrInvalidElementType
	}
}

func (e ImageElement) validate() error {
	if e.Type != ImageElementType {
		return ErrInvalidElementType
	}

	if e.ImageURL == "" {
		return ErrNilImageURL
	}

	if e.AltText == "" {
		return ErrNilAltText
	}

	return nil
}

func (ImageElement) sectionAccessory() {}

func (ImageElement) contextElement() {}

func (b Button) validate() error {
	if b.Type != ButtonElement {
		return ErrInvalidElementType
	}

	if b.Text == nil {
		return ErrNilText
	}

	if err := b.Text.validate(PlainText); err != nil {
		return err
	}

	switch b.Style {
	case "", PrimaryButton, DangerButton:
		return nil
	default:
		return ErrInvalidButtonStyle
	}
}

func (Button) sectionAccessory() {}

func (Button) actionElement() {}

func (ContextText) contextElement() {}

func containsTextType(types []TextType, textType TextType) bool {
	for _, typ := range types {
		if typ == textType {
			return true
		}
	}

	return false
}

func boolPtr(value bool) *bool {
	return &value
}

func textPtr(value Text) *Text {
	return &value
}
