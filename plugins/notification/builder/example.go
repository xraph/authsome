package builder

// Example demonstrates how to use the email builder

// ExampleCreateTemplate shows how to create a template from scratch.
func ExampleCreateTemplate() {
	// Create a new document
	doc := NewDocument()

	// Add a spacer
	_, _ = doc.AddBlock(BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"height": 20,
		},
	}, doc.Root)

	// Add a heading
	_, _ = doc.AddBlock(BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]any{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]any{
			"text":  "Welcome to Our Platform!",
			"level": "h1",
		},
	}, doc.Root)

	// Add text content
	_, _ = doc.AddBlock(BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":  15,
			"textAlign": "center",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Thank you for joining us. We're excited to have you on board!",
		},
	}, doc.Root)

	// Add a button
	_, _ = doc.AddBlock(BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 16, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"text":         "Get Started",
			"url":          "https://example.com/dashboard",
			"buttonColor":  "#0066CC",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
		},
	}, doc.Root)

	// Render to HTML
	renderer := NewRenderer(doc)

	_, err := renderer.RenderToHTML()
	if err != nil {
		return
	}

	// Get JSON representation
	_, err = doc.ToJSON()
	if err != nil {
		return
	}
}

// ExampleUseSampleTemplate shows how to use a sample template.
func ExampleUseSampleTemplate() {
	// Load a sample template
	doc, err := GetSampleTemplate("welcome")
	if err != nil {
		return
	}

	// Render with variables
	_, err = RenderTemplate(doc, map[string]any{
		"AppName":      "My Awesome App",
		"UserName":     "John Doe",
		"DashboardURL": "https://example.com/dashboard",
	})
	if err != nil {
		return
	}
}

// ExampleModifyTemplate shows how to modify an existing template.
func ExampleModifyTemplate() {
	// Load template
	doc, err := GetSampleTemplate("otp")
	if err != nil {
		return
	}

	// Find and modify the OTP block
	for blockID, block := range doc.Blocks {
		if block.Type == BlockTypeText {
			if props, ok := block.Data["props"].(map[string]any); ok {
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

	_, err = renderer.RenderToHTML()
	if err != nil {
		return
	}
}

// ExampleComplexLayout shows how to create a complex layout with columns.
func ExampleComplexLayout() {
	doc := NewDocument()

	// Add heading
	_, _ = doc.AddBlock(BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"padding": map[string]any{
				"top": 24, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]any{
			"text":  "Product Showcase",
			"level": "h2",
		},
	}, doc.Root)

	// Add columns
	columnsID, _ := doc.AddBlock(BlockTypeColumns, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]any{
			"columnsCount": 2,
			"columnsGap":   16,
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Add first column
	col1ID, _ := doc.AddBlock(BlockTypeColumn, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"width": "50%",
		},
		"childrenIds": []string{},
	}, columnsID)

	// Add image to first column
	_, _ = doc.AddBlock(BlockTypeImage, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"url":   "https://via.placeholder.com/300x200",
			"alt":   "Product 1",
			"width": "100%",
		},
	}, col1ID)

	// Add text to first column
	_, _ = doc.AddBlock(BlockTypeText, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 12, "right": 8, "bottom": 8, "left": 8,
			},
		},
		"props": map[string]any{
			"text": "<strong>Premium Plan</strong><br/>$29/month",
		},
	}, col1ID)

	// Add second column
	col2ID, _ := doc.AddBlock(BlockTypeColumn, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"width": "50%",
		},
		"childrenIds": []string{},
	}, columnsID)

	// Add image to second column
	_, _ = doc.AddBlock(BlockTypeImage, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"url":   "https://via.placeholder.com/300x200",
			"alt":   "Product 2",
			"width": "100%",
		},
	}, col2ID)

	// Add text to second column
	_, _ = doc.AddBlock(BlockTypeText, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 12, "right": 8, "bottom": 8, "left": 8,
			},
		},
		"props": map[string]any{
			"text": "<strong>Enterprise Plan</strong><br/>$99/month",
		},
	}, col2ID)

	// Render
	renderer := NewRenderer(doc)

	_, err := renderer.RenderToHTML()
	if err != nil {
		return
	}
}

// ExampleValidation shows how to validate a document.
func ExampleValidation() {
	// Create a document
	doc := NewDocument()

	// Add some blocks
	_, _ = doc.AddBlock(BlockTypeText, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"text": "Hello World",
		},
	}, doc.Root)

	// Validate
	if err := doc.Validate(); err != nil {
		return
	}

	// Try to add invalid block reference
	doc.Blocks["root"].Data["childrenIds"] = []any{"non-existent-block"}

	if err := doc.Validate(); err != nil {

	}
}

// ExampleSerialization shows how to save and load documents.
func ExampleSerialization() {
	// Create a document
	doc := NewDocument()
	_, _ = doc.AddBlock(BlockTypeHeading, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{
			"text":  "Test Heading",
			"level": "h2",
		},
	}, doc.Root)

	// Convert to JSON
	jsonStr, err := doc.ToJSON()
	if err != nil {
		return
	}

	// Load from JSON
	_, err = FromJSON(jsonStr)
	if err != nil {
		return
	}
}

// RunAllExamples runs all example functions.
func RunAllExamples() {
	ExampleCreateTemplate()

	ExampleUseSampleTemplate()

	ExampleModifyTemplate()

	ExampleComplexLayout()

	ExampleValidation()

	ExampleSerialization()
}
