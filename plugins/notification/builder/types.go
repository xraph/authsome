package builder

import (
	"encoding/json"
	"fmt"

	"github.com/xraph/authsome/internal/errs"
)

// BlockType represents the type of email block.
type BlockType string

const (
	BlockTypeEmailLayout BlockType = "EmailLayout"
	BlockTypeText        BlockType = "Text"
	BlockTypeHeading     BlockType = "Heading"
	BlockTypeButton      BlockType = "Button"
	BlockTypeImage       BlockType = "Image"
	BlockTypeDivider     BlockType = "Divider"
	BlockTypeSpacer      BlockType = "Spacer"
	BlockTypeContainer   BlockType = "Container"
	BlockTypeColumns     BlockType = "Columns"
	BlockTypeColumn      BlockType = "Column"
	BlockTypeHTML        BlockType = "HTML"
	BlockTypeAvatar      BlockType = "Avatar"
)

// Document represents the email builder document structure.
type Document struct {
	Root   string           `json:"root"`
	Blocks map[string]Block `json:"blocks"`
}

// Block represents a single block in the email.
type Block struct {
	Type BlockType      `json:"type"`
	Data map[string]any `json:"data"`
}

// EmailLayoutData represents the root email layout configuration.
type EmailLayoutData struct {
	BackdropColor string   `json:"backdropColor"`
	CanvasColor   string   `json:"canvasColor"`
	TextColor     string   `json:"textColor"`
	LinkColor     string   `json:"linkColor"`
	FontFamily    string   `json:"fontFamily"`
	ChildrenIDs   []string `json:"childrenIds"`
}

// Padding represents padding configuration.
type Padding struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// BlockStyle represents common style properties for blocks.
type BlockStyle struct {
	BackgroundColor string   `json:"backgroundColor,omitempty"`
	Color           string   `json:"color,omitempty"`
	FontFamily      string   `json:"fontFamily,omitempty"`
	FontSize        int      `json:"fontSize,omitempty"`
	FontWeight      string   `json:"fontWeight,omitempty"`
	TextAlign       string   `json:"textAlign,omitempty"`
	Padding         *Padding `json:"padding,omitempty"`
}

// TextBlockData represents text block configuration.
type TextBlockData struct {
	Style BlockStyle     `json:"style"`
	Props TextBlockProps `json:"props"`
}

type TextBlockProps struct {
	Text string `json:"text"`
}

// HeadingBlockData represents heading block configuration.
type HeadingBlockData struct {
	Style BlockStyle        `json:"style"`
	Props HeadingBlockProps `json:"props"`
}

type HeadingBlockProps struct {
	Text  string `json:"text"`
	Level string `json:"level"` // h1, h2, h3, h4, h5, h6
}

// ButtonBlockData represents button block configuration.
type ButtonBlockData struct {
	Style BlockStyle       `json:"style"`
	Props ButtonBlockProps `json:"props"`
}

type ButtonBlockProps struct {
	Text         string `json:"text"`
	URL          string `json:"url"`
	ButtonColor  string `json:"buttonColor"`
	TextColor    string `json:"textColor"`
	BorderRadius int    `json:"borderRadius"`
	FullWidth    bool   `json:"fullWidth"`
}

// ImageBlockData represents image block configuration.
type ImageBlockData struct {
	Style BlockStyle      `json:"style"`
	Props ImageBlockProps `json:"props"`
}

type ImageBlockProps struct {
	URL              string `json:"url"`
	Alt              string `json:"alt"`
	LinkURL          string `json:"linkUrl,omitempty"`
	Width            string `json:"width"`
	Height           string `json:"height,omitempty"`
	ContentAlignment string `json:"contentAlignment"` // left, center, right
}

// DividerBlockData represents divider block configuration.
type DividerBlockData struct {
	Style BlockStyle        `json:"style"`
	Props DividerBlockProps `json:"props"`
}

type DividerBlockProps struct {
	LineColor  string `json:"lineColor"`
	LineHeight int    `json:"lineHeight"`
}

// SpacerBlockData represents spacer block configuration.
type SpacerBlockData struct {
	Style BlockStyle       `json:"style"`
	Props SpacerBlockProps `json:"props"`
}

type SpacerBlockProps struct {
	Height int `json:"height"`
}

// ContainerBlockData represents container block configuration.
type ContainerBlockData struct {
	Style       BlockStyle          `json:"style"`
	Props       ContainerBlockProps `json:"props"`
	ChildrenIDs []string            `json:"childrenIds"`
}

type ContainerBlockProps struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

// ColumnsBlockData represents columns block configuration.
type ColumnsBlockData struct {
	Style       BlockStyle        `json:"style"`
	Props       ColumnsBlockProps `json:"props"`
	ChildrenIDs []string          `json:"childrenIds"` // Column IDs
}

type ColumnsBlockProps struct {
	ColumnsCount int `json:"columnsCount"`
	ColumnsGap   int `json:"columnsGap"`
}

// ColumnBlockData represents a single column in columns block.
type ColumnBlockData struct {
	Style       BlockStyle       `json:"style"`
	Props       ColumnBlockProps `json:"props"`
	ChildrenIDs []string         `json:"childrenIds"`
}

type ColumnBlockProps struct {
	Width string `json:"width,omitempty"` // Can be percentage or auto
}

// HTMLBlockData represents raw HTML block configuration.
type HTMLBlockData struct {
	Style BlockStyle     `json:"style"`
	Props HTMLBlockProps `json:"props"`
}

type HTMLBlockProps struct {
	HTML string `json:"html"`
}

// AvatarBlockData represents avatar block configuration.
type AvatarBlockData struct {
	Style BlockStyle       `json:"style"`
	Props AvatarBlockProps `json:"props"`
}

type AvatarBlockProps struct {
	ImageURL string `json:"imageUrl"`
	Alt      string `json:"alt"`
	Size     int    `json:"size"`
	Shape    string `json:"shape"` // circle, square, rounded
}

// Helper functions to create blocks

// NewDocument creates a new empty email document.
func NewDocument() *Document {
	rootID := "root"

	return &Document{
		Root: rootID,
		Blocks: map[string]Block{
			rootID: {
				Type: BlockTypeEmailLayout,
				Data: map[string]any{
					"backdropColor": "#F8F8F8",
					"canvasColor":   "#FFFFFF",
					"textColor":     "#242424",
					"linkColor":     "#0066CC",
					"fontFamily":    "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
					"childrenIds":   []string{},
				},
			},
		},
	}
}

// AddBlock adds a block to the document and returns its ID.
func (d *Document) AddBlock(blockType BlockType, data map[string]any, parentID string) (string, error) {
	// Generate unique block ID
	blockID := fmt.Sprintf("block-%d", len(d.Blocks))

	// Add the block
	d.Blocks[blockID] = Block{
		Type: blockType,
		Data: data,
	}

	// Add to parent's children if parent exists
	if parentID != "" {
		parent, exists := d.Blocks[parentID]
		if !exists {
			return "", fmt.Errorf("parent block %s not found", parentID)
		}

		// Get or create childrenIds
		childrenIDs, ok := parent.Data["childrenIds"].([]string)
		if !ok {
			childrenIDs = []string{}
		}

		childrenIDs = append(childrenIDs, blockID)
		parent.Data["childrenIds"] = childrenIDs
		d.Blocks[parentID] = parent
	}

	return blockID, nil
}

// RemoveBlock removes a block from the document.
func (d *Document) RemoveBlock(blockID string) error {
	if blockID == d.Root {
		return errs.BadRequest("cannot remove root block")
	}

	// Remove from parent's children
	for id, block := range d.Blocks {
		if childrenIDs, ok := block.Data["childrenIds"].([]string); ok {
			for i, childID := range childrenIDs {
				if childID == blockID {
					childrenIDs = append(childrenIDs[:i], childrenIDs[i+1:]...)
					block.Data["childrenIds"] = childrenIDs
					d.Blocks[id] = block

					break
				}
			}
		}
	}

	// Remove the block itself
	delete(d.Blocks, blockID)

	return nil
}

// ToJSON converts the document to JSON string.
func (d *Document) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	return string(bytes), nil
}

// FromJSON creates a document from JSON string.
func FromJSON(jsonStr string) (*Document, error) {
	var doc Document
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &doc, nil
}

// Validate validates the document structure.
func (d *Document) Validate() error {
	// Check if root exists
	if _, exists := d.Blocks[d.Root]; !exists {
		return errs.NotFound("root block not found")
	}

	// Validate all blocks
	for id, block := range d.Blocks {
		if block.Type == "" {
			return fmt.Errorf("block %s has no type", id)
		}

		// Validate children exist
		if childrenIDs, ok := block.Data["childrenIds"].([]string); ok {
			for _, childID := range childrenIDs {
				if _, exists := d.Blocks[childID]; !exists {
					return fmt.Errorf("block %s references non-existent child %s", id, childID)
				}
			}
		}
	}

	return nil
}
