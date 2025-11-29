# AuthSome Dashboard Frontend

This directory contains the frontend build pipeline for the AuthSome Dashboard, using Tailwind CSS v4 and Preline UI.

## Prerequisites

- Node.js 18+ 
- npm 9+

## Quick Start

```bash
# Install dependencies
npm install

# Build CSS and JS
npm run build

# Watch for CSS changes (development)
npm run watch
```

## Build Commands

| Command | Description |
|---------|-------------|
| `npm run build` | Build both CSS and JS bundles |
| `npm run build:css` | Build only Tailwind CSS |
| `npm run build:js` | Build only JavaScript bundle |
| `npm run watch` | Watch CSS for changes |

## Output Files

The build generates:

- `../static/css/dashboard.css` - Compiled Tailwind CSS + Preline UI styles
- `../static/js/bundle.js` - Combined Preline + custom Alpine.js components

These files are embedded into the Go binary via `go:embed`.

## Stack

- **[Tailwind CSS v4](https://tailwindcss.com/)** - Utility-first CSS framework
- **[Preline UI v3](https://preline.co/)** - Tailwind CSS component library
- **[Alpine.js](https://alpinejs.dev/)** - Lightweight JavaScript framework (loaded via CDN)
- **[esbuild](https://esbuild.github.io/)** - Fast JavaScript bundler

## Using Preline Components

Preline UI provides interactive components that work with JavaScript. After the build, you can use Preline components in your Go templates.

### Example: Accordion

```html
<div class="hs-accordion-group">
  <div class="hs-accordion active" id="hs-basic-heading-one">
    <button class="hs-accordion-toggle" aria-controls="hs-basic-collapse-one">
      Accordion #1
    </button>
    <div id="hs-basic-collapse-one" class="hs-accordion-content">
      Content here...
    </div>
  </div>
</div>
```

### Example: Modal

```html
<button type="button" data-hs-overlay="#modal-id">Open Modal</button>

<div id="modal-id" class="hs-overlay hidden">
  <div class="hs-overlay-open:mt-7 hs-overlay-open:opacity-100">
    <!-- Modal content -->
  </div>
</div>
```

### Example: Dropdown

```html
<div class="hs-dropdown relative inline-flex">
  <button type="button" class="hs-dropdown-toggle">
    Dropdown
  </button>
  <div class="hs-dropdown-menu hidden">
    <a href="#">Item 1</a>
    <a href="#">Item 2</a>
  </div>
</div>
```

See [Preline UI Documentation](https://preline.co/docs/) for all available components.

## Makefile Targets

From the project root, you can use:

```bash
# Install dependencies
make dashboard-setup

# Build assets
make dashboard-build

# Watch for changes
make dashboard-watch

# Clean and rebuild
make dashboard-rebuild

# Clean artifacts
make dashboard-clean
```

## Development Workflow

1. Make changes to `src/input.css` for custom styles
2. Run `npm run watch` to auto-rebuild on changes
3. Refresh the browser to see changes

For JavaScript changes:
1. Edit `../static/js/pines-components.js` or `../static/js/dashboard.js`
2. Run `npm run build:js` to rebuild the bundle

## Customization

### Adding Custom Styles

Edit `src/input.css` to add custom Tailwind utilities or components:

```css
@layer components {
  .btn-custom {
    @apply px-4 py-2 rounded-lg bg-primary text-white;
  }
}
```

### Extending Theme

Modify the `@theme` section in `src/input.css`:

```css
@theme {
  --color-accent-500: oklch(0.7 0.2 250);
}
```

## Troubleshooting

### Build fails with "Can't resolve preline/variants.css"

Make sure you have Preline v3.x installed:

```bash
npm install preline@^3.2.0
```

### Styles not applying

1. Ensure `dashboard.css` is being served correctly
2. Check browser console for 404 errors
3. Verify the Go embed directive includes the CSS file

### JavaScript components not working

1. Ensure `bundle.js` is loaded before Alpine.js
2. Check browser console for errors
3. Run `npm run build:js` to rebuild

