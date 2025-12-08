package builder

import (
	"fmt"
)

// Example demonstrates how to use the email builder

// ExampleCreateTemplate shows how to create a template from scratch
func ExampleCreateTemplate() {
	// Create a new document
	doc := NewDocument()

	// Add a spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add a heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Welcome to Our Platform!",
			"level": "h1",
		},
	}, doc.Root)

	// Add text content
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":  15,
			"textAlign": "center",
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Thank you for joining us. We're excited to have you on board!",
		},
	}, doc.Root)

	// Add a button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "Get Started",
			"url":          "https://example.com/dashboard",
			"buttonColor":  "#0066CC",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
		},
	}, doc.Root)

	// Render to HTML
	renderer := NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		fmt.Printf("Error rendering: %v\n", err)
		return
	}

	fmt.Printf("Generated HTML (%d bytes)\n", len(html))

	// Get JSON representation
	jsonStr, err := doc.ToJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		return
	}

	fmt.Printf("JSON document (%d bytes)\n", len(jsonStr))
}

// ExampleUseSampleTemplate shows how to use a sample template
func ExampleUseSampleTemplate() {
	// Load a sample template
	doc, err := GetSampleTemplate("welcome")
	if err != nil {
		fmt.Printf("Error loading template: %v\n", err)
		return
	}

	// Render with variables
	html, err := RenderTemplate(doc, map[string]interface{}{
		"AppName":      "My Awesome App",
		"UserName":     "John Doe",
		"DashboardURL": "https://example.com/dashboard",
	})

	if err != nil {
		fmt.Printf("Error rendering: %v\n", err)
		return
	}

	fmt.Printf("Rendered email with variables (%d bytes)\n", len(html))
}

// ExampleModifyTemplate shows how to modify an existing template
func ExampleModifyTemplate() {
	// Load template
	doc, err := GetSampleTemplate("otp")
	if err != nil {
		fmt.Printf("Error loading template: %v\n", err)
		return
	}

	// Find and modify the OTP block
	for blockID, block := range doc.Blocks {
		if block.Type == BlockTypeText {
			if props, ok := block.Data["props"].(map[string]interface{}); ok {
				if text, ok := props["text"].(string); ok {
					// Modify the text
					props["text"] = text + " (Modified)"
					block.Data["props"] = props
					doc.Blocks[blockID] = block
				}
			}
		}
	}

	// Render modified template
	renderer := NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		fmt.Printf("Error rendering: %v\n", err)
		return
	}

	fmt.Printf("Modified template rendered (%d bytes)\n", len(html))
}

// ExampleComplexLayout shows how to create a complex layout with columns
func ExampleComplexLayout() {
	doc := NewDocument()

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"padding": map[string]interface{}{
				"top": 24, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Product Showcase",
			"level": "h2",
		},
	}, doc.Root)

	// Add columns
	columnsID, _ := doc.AddBlock(BlockTypeColumns, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"columnsCount": 2,
			"columnsGap":   16,
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Add first column
	col1ID, _ := doc.AddBlock(BlockTypeColumn, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"width": "50%",
		},
		"childrenIds": []string{},
	}, columnsID)

	// Add image to first column
	doc.AddBlock(BlockTypeImage, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"url":   "https://via.placeholder.com/300x200",
			"alt":   "Product 1",
			"width": "100%",
		},
	}, col1ID)

	// Add text to first column
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 12, "right": 8, "bottom": 8, "left": 8,
			},
		},
		"props": map[string]interface{}{
			"text": "<strong>Premium Plan</strong><br/>$29/month",
		},
	}, col1ID)

	// Add second column
	col2ID, _ := doc.AddBlock(BlockTypeColumn, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"width": "50%",
		},
		"childrenIds": []string{},
	}, columnsID)

	// Add image to second column
	doc.AddBlock(BlockTypeImage, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"url":   "https://via.placeholder.com/300x200",
			"alt":   "Product 2",
			"width": "100%",
		},
	}, col2ID)

	// Add text to second column
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 12, "right": 8, "bottom": 8, "left": 8,
			},
		},
		"props": map[string]interface{}{
			"text": "<strong>Enterprise Plan</strong><br/>$99/month",
		},
	}, col2ID)

	// Render
	renderer := NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		fmt.Printf("Error rendering: %v\n", err)
		return
	}

	fmt.Printf("Complex layout rendered (%d bytes)\n", len(html))
}

// ExampleValidation shows how to validate a document
func ExampleValidation() {
	// Create a document
	doc := NewDocument()

	// Add some blocks
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"text": "Hello World",
		},
	}, doc.Root)

	// Validate
	if err := doc.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	fmt.Println("Document is valid!")

	// Try to add invalid block reference
	doc.Blocks["root"].Data["childrenIds"] = []interface{}{"non-existent-block"}

	if err := doc.Validate(); err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}
}

// ExampleSerialization shows how to save and load documents
func ExampleSerialization() {
	// Create a document
	doc := NewDocument()
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"text":  "Test Heading",
			"level": "h2",
		},
	}, doc.Root)

	// Convert to JSON
	jsonStr, err := doc.ToJSON()
	if err != nil {
		fmt.Printf("Error serializing: %v\n", err)
		return
	}

	fmt.Printf("Serialized document: %d bytes\n", len(jsonStr))

	// Load from JSON
	loadedDoc, err := FromJSON(jsonStr)
	if err != nil {
		fmt.Printf("Error deserializing: %v\n", err)
		return
	}

	fmt.Printf("Loaded document with %d blocks\n", len(loadedDoc.Blocks))
}

// RunAllExamples runs all example functions
func RunAllExamples() {
	fmt.Println("=== Example: Create Template ===")
	ExampleCreateTemplate()

	fmt.Println("\n=== Example: Use Sample Template ===")
	ExampleUseSampleTemplate()

	fmt.Println("\n=== Example: Modify Template ===")
	ExampleModifyTemplate()

	fmt.Println("\n=== Example: Complex Layout ===")
	ExampleComplexLayout()

	fmt.Println("\n=== Example: Validation ===")
	ExampleValidation()

	fmt.Println("\n=== Example: Serialization ===")
	ExampleSerialization()
}
