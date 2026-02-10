package cms

import (
	"strconv"
	"strings"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/router"
)

// parseFieldOptions extracts field options from the form.
func (e *DashboardExtension) parseFieldOptions(c forge.Context) *core.FieldOptionsDTO {
	opts := &core.FieldOptionsDTO{}

	// Text options
	if v := c.FormValue("options.minLength"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			opts.MinLength = i
		}
	}

	if v := c.FormValue("options.maxLength"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			opts.MaxLength = i
		}
	}

	opts.Pattern = c.FormValue("options.regex")
	if v := c.FormValue("options.default"); v != "" {
		// Store default value based on type? For now just string if it's text
		// The logic for default value type conversion might be needed later in service
	}

	// Number options
	if v := c.FormValue("options.min"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Min = &f
		}
	}

	if v := c.FormValue("options.max"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Max = &f
		}
	}

	if v := c.FormValue("options.step"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Step = &f
		}
	}

	// Enums (Choices)
	// Iterate through form values to find options.enum[i].value/label
	choicesMap := make(map[int]core.ChoiceDTO)
	// We need to parse the raw form values because FormValue only gets the first one
	// and we don't know the indices ahead of time easily
	if c.Request().PostForm == nil {
		_ = c.Request().ParseForm() // Ignore parse errors
	}

	for key, values := range c.Request().PostForm {
		if len(values) == 0 {
			continue
		}

		val := values[0]

		// Check for options.enum[index].field
		if strings.HasPrefix(key, "options.enum[") {
			// Extract index and field
			// Format: options.enum[0].label
			parts := strings.Split(key, ".")
			if len(parts) < 3 {
				continue
			}

			// Extract index from enum[0]
			indexPart := parts[1]
			if len(indexPart) < 6 { // enum[ + ]
				continue
			}

			indexStr := indexPart[5 : len(indexPart)-1]

			index, err := strconv.Atoi(indexStr)
			if err != nil {
				continue
			}

			field := parts[2] // label or value

			choice := choicesMap[index]

			switch field {
			case "label":
				choice.Label = val
			case "value":
				choice.Value = val
			}

			choicesMap[index] = choice
		}
	}

	// Convert map to slice, ordered by index
	if len(choicesMap) > 0 {
		// Find max index
		maxIdx := -1
		for k := range choicesMap {
			if k > maxIdx {
				maxIdx = k
			}
		}

		for i := 0; i <= maxIdx; i++ {
			if choice, ok := choicesMap[i]; ok {
				// If label is empty but value is set, use value as label
				if choice.Label == "" && choice.Value != "" {
					choice.Label = choice.Value
				}

				if choice.Value != "" { // Only add if value exists
					opts.Choices = append(opts.Choices, choice)
				}
			}
		}
	}

	return opts
}

// parseFieldOptionsFromRequest extracts field options from the router.PageContext.
func (e *DashboardExtension) parseFieldOptionsFromRequest(ctx *router.PageContext) *core.FieldOptionsDTO {
	opts := &core.FieldOptionsDTO{}

	// Parse form if needed
	if ctx.Request.PostForm == nil {
		_ = ctx.Request.ParseForm()
	}

	// Text options
	if v := ctx.Request.FormValue("options.minLength"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			opts.MinLength = i
		}
	}

	if v := ctx.Request.FormValue("options.maxLength"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			opts.MaxLength = i
		}
	}

	opts.Pattern = ctx.Request.FormValue("options.regex")

	// Number options
	if v := ctx.Request.FormValue("options.min"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Min = &f
		}
	}

	if v := ctx.Request.FormValue("options.max"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Max = &f
		}
	}

	if v := ctx.Request.FormValue("options.step"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.Step = &f
		}
	}

	// Enums (Choices)
	choicesMap := make(map[int]core.ChoiceDTO)

	for key, values := range ctx.Request.PostForm {
		if len(values) == 0 {
			continue
		}

		val := values[0]

		// Check for options.enum[index].field
		if strings.HasPrefix(key, "options.enum[") {
			// Extract index and field
			// Format: options.enum[0].label
			parts := strings.Split(key, ".")
			if len(parts) < 3 {
				continue
			}

			// Extract index from enum[0]
			indexPart := parts[1]
			if len(indexPart) < 6 { // enum[ + ]
				continue
			}

			indexStr := indexPart[5 : len(indexPart)-1]

			index, err := strconv.Atoi(indexStr)
			if err != nil {
				continue
			}

			field := parts[2] // label or value

			choice := choicesMap[index]

			switch field {
			case "label":
				choice.Label = val
			case "value":
				choice.Value = val
			}

			choicesMap[index] = choice
		}
	}

	// Convert map to slice, ordered by index
	if len(choicesMap) > 0 {
		// Find max index
		maxIdx := -1
		for k := range choicesMap {
			if k > maxIdx {
				maxIdx = k
			}
		}

		for i := 0; i <= maxIdx; i++ {
			if choice, ok := choicesMap[i]; ok {
				// If label is empty but value is set, use value as label
				if choice.Label == "" && choice.Value != "" {
					choice.Label = choice.Value
				}

				if choice.Value != "" { // Only add if value exists
					opts.Choices = append(opts.Choices, choice)
				}
			}
		}
	}

	return opts
}
