# Email Template Builder

A beautiful, minimalist email template builder for AuthSome, inspired by [email-builder-js](https://github.com/usewaypoint/email-builder-js).

## Features

- 🎨 **Visual Drag & Drop Editor** - Build emails visually without writing HTML
- 📱 **Mobile Responsive** - All templates are mobile-friendly by default
- 🧱 **Block-Based** - Compose emails from reusable blocks
- 👁️ **Live Preview** - See changes in real-time
- 💾 **JSON Storage** - Templates stored as clean JSON configuration
- 🎯 **Variable Support** - Use template variables like `{{.UserName}}`
- 📦 **Sample Templates** - Pre-built templates to get started quickly
- 🔧 **Extensible** - Easy to add custom blocks

## Architecture

The builder is composed of several key components:

### 1. Block Types (`types.go`)

Defines all available block types and their data structures:

- **EmailLayout** - Root container with global styles
- **Text** - Paragraph text with formatting
- **Heading** - Section headings (h1-h6)
- **Button** - Call-to-action buttons
- **Image** - Pictures and logos
- **Divider** - Horizontal separators
- **Spacer** - Vertical spacing
- **Container** - Group blocks together
- **Columns** - Multi-column layouts
- **Avatar** - Profile pictures
- **HTML** - Custom HTML code

### 2. Renderer (`renderer.go`)

Converts JSON documents to HTML email:

```go
doc := NewDocument()
renderer := NewRenderer(doc)
html, err := renderer.RenderToHTML()
```

The renderer uses gomponents to generate email-safe HTML that works across all email clients.

### 3. Templates (`templates.go`)

Pre-built sample templates:

- **Welcome Email** - Greet new users
- **OTP Code** - One-time password verification
- **Password Reset** - Secure password reset
- **Invitation** - Organization invitations
- **Notification** - General purpose notifications

### 4. UI (`ui.go`)

Visual builder interface with:

- **Left Sidebar** - Block palette and sample templates
- **Center Canvas** - Design, preview, and code views
- **Right Sidebar** - Properties panel for selected blocks

### 5. Dashboard Integration (`dashboard.go`)

Integrates the builder into the AuthSome dashboard:

- Builder routes
- Template management
- Preview generation
- Sample template loading

## Usage

### Creating a New Template

```go
// Create a new empty document
doc := builder.NewDocument()

// Add blocks
blockID, _ := doc.AddBlock(builder.BlockTypeHeading, map[string]interface{}{
    "style": map[string]interface{}{
        "textAlign": "center",
        "color": "#1a1a1a",
    },
    "props": map[string]interface{}{
        "text": "Welcome!",
        "level": "h1",
    },
}, doc.Root)

// Render to HTML
renderer := builder.NewRenderer(doc)
html, _ := renderer.RenderToHTML()
```

### Using Sample Templates

```go
// Load a sample template
doc, err := builder.GetSampleTemplate("welcome")

// Customize it
// ... modify blocks ...

// Render with variables
html, err := builder.RenderTemplate(doc, map[string]interface{}{
    "AppName": "My App",
    "UserName": "John Doe",
    "DashboardURL": "https://example.com/dashboard",
})
```

### Integrating with Dashboard

```go
// In your plugin initialization
builderHandler := builder.NewDashboardHandler(notificationService)
routes := builderHandler.RegisterRoutes()

// Register routes with dashboard
for _, route := range routes {
    dashboardPlugin.RegisterRoute(route)
}
```

## Block Structure

Each block in the document has this structure:

```json
{
  "type": "BlockType",
  "data": {
    "style": {
      "color": "#333",
      "fontSize": 16,
      "padding": {
        "top": 16,
        "right": 24,
        "bottom": 16,
        "left": 24
      }
    },
    "props": {
      // Block-specific properties
    },
    "childrenIds": [] // For container blocks
  }
}
```

## Email Client Compatibility

All blocks are tested and compatible with:

- ✅ Gmail (Desktop & Mobile)
- ✅ Apple Mail (Desktop & Mobile)
- ✅ Outlook (2016+, Office 365, Web)
- ✅ Yahoo! Mail
- ✅ Thunderbird
- ✅ Mobile clients (iOS Mail, Android Gmail)

## Template Variables

The builder supports Go template variables:

```html
Hello {{.UserName}},

Welcome to {{.AppName}}!

{{if .HasPremium}}
You have premium access!
{{end}}
```

Variables are rendered using Go's `text/template` engine in the notification service.

## Extending the Builder

### Adding a Custom Block

1. Define the block type in `types.go`:

```go
const BlockTypeCustom BlockType = "Custom"

type CustomBlockData struct {
    Style Style              `json:"style"`
    Props CustomBlockProps   `json:"props"`
}

type CustomBlockProps struct {
    CustomField string `json:"customField"`
}
```

2. Add renderer in `renderer.go`:

```go
func (r *Renderer) renderCustom(block Block) g.Node {
    data := block.Data
    props := getMap(data, "props")
    
    customField := getString(props, "customField", "")
    
    return Div(
        g.Text(customField),
    )
}
```

3. Update the UI block palette in `ui.go`

4. Add default data in `getDefaultBlockData()`

## Best Practices

### Design Guidelines

1. **Keep it Simple** - Use clear hierarchy and whitespace
2. **Mobile First** - Test on mobile devices
3. **Readable Typography** - Use 14-16px for body text
4. **Clear CTAs** - Make buttons obvious and actionable
5. **Brand Consistency** - Use consistent colors and fonts

### Technical Guidelines

1. **Use Tables for Layout** - Email clients require table-based layouts
2. **Inline Styles** - The renderer uses inline styles for compatibility
3. **Test Thoroughly** - Test across different email clients
4. **Keep Images Small** - Optimize images for fast loading
5. **Alt Text** - Always provide alt text for images

## Performance

- **Lightweight** - Minimal JavaScript, no external dependencies
- **Fast Rendering** - Gomponents generates HTML efficiently
- **Small Payloads** - JSON documents are compact
- **Cacheable** - Templates can be cached once rendered

## Future Enhancements

- [ ] Undo/Redo functionality
- [ ] Copy/paste blocks
- [ ] Block templates library
- [ ] A/B testing support
- [ ] Dark mode support
- [ ] Export to other formats (MJML, React Email)
- [ ] Advanced layout blocks (grids, carousels)
- [ ] Custom CSS support
- [ ] Template marketplace

## Credits

Inspired by:

- [email-builder-js](https://github.com/usewaypoint/email-builder-js) by Waypoint
- [React Email](https://react.email/)
- [MJML](https://mjml.io/)

Built with:

- [gomponents](https://www.gomponents.com/) for HTML generation
- [Alpine.js](https://alpinejs.dev/) for interactivity
- [Lucide Icons](https://lucide.dev/) for beautiful icons

## License

Part of AuthSome - see main project LICENSE

