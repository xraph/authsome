# Dashboard Plugin

The Dashboard Plugin provides a modern React-based administrative interface for AuthSome. It offers a comprehensive web UI for managing users, sessions, security settings, and monitoring authentication activities.

## Features

- 📊 **User Management**: View, create, edit, and manage user accounts
- 🔐 **Session Monitoring**: Real-time session tracking and management
- ⚙️ **Security Settings**: Configure authentication policies and security rules
- 🔌 **Plugin Management**: Enable/disable and configure authentication plugins
- 📈 **Analytics Dashboard**: Authentication metrics and usage statistics
- 🎨 **Modern UI**: Built with React, TypeScript, and Tailwind CSS
- 📱 **Responsive Design**: Works on desktop, tablet, and mobile devices

## Installation

The dashboard plugin is included with AuthSome. To use it, simply register it with your AuthSome instance:

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/dashboard"
)

// Create AuthSome instance
auth := authsome.New(
    authsome.WithMode(authsome.ModeStandalone),
)

// Register the dashboard plugin
err := auth.RegisterPlugin(dashboard.NewPlugin())
if err != nil {
    log.Fatal("Failed to register dashboard plugin:", err)
}

// Initialize and mount
err = auth.Initialize(context.Background())
if err != nil {
    log.Fatal("Failed to initialize AuthSome:", err)
}

err = auth.Mount(app, "/api/auth")
if err != nil {
    log.Fatal("Failed to mount AuthSome:", err)
}
```

## Usage

Once registered and mounted, the dashboard will be available at:

```
http://localhost:PORT/dashboard
```

### Routes

The plugin registers the following routes:

- `GET /dashboard/` - Main dashboard SPA entry point
- `GET /dashboard/*` - Static assets (JS, CSS, images, etc.)

### Assets

The dashboard includes embedded static assets built from the React application:

```go
// Access embedded assets programmatically
assets := dashboard.GetAssets()
indexContent, err := fs.ReadFile(assets, "index.html")
```

## Development

### Frontend Development

The dashboard frontend is located in `frontend/dashboard/` and built with:

- **React 18** - Modern React with hooks and concurrent features
- **TypeScript** - Type-safe JavaScript development
- **Vite** - Fast build tool and development server
- **Tailwind CSS** - Utility-first CSS framework
- **Lucide React** - Beautiful icon library

#### Development Setup

```bash
# Navigate to frontend directory
cd frontend/dashboard

# Install dependencies
pnpm install

# Start development server
pnpm dev

# Build for production
pnpm build
```

#### Development Server

The development server runs on `http://localhost:5173` with:

- Hot module replacement (HMR)
- TypeScript checking
- ESLint integration
- Automatic browser refresh

### Plugin Development

The plugin follows the standard AuthSome plugin interface:

```go
type Plugin interface {
    ID() string
    Init(dep interface{}) error
    RegisterRoutes(router interface{}) error
    RegisterHooks(hooks *hooks.HookRegistry) error
    RegisterServiceDecorators(services *registry.ServiceRegistry) error
    Migrate() error
}
```

#### Plugin Structure

```
plugins/dashboard/
├── plugin.go              # Main plugin implementation
├── handler.go              # HTTP handlers for serving assets
├── plugin_test.go          # Plugin tests
├── README.md               # This documentation
└── dist/                   # Built frontend assets (embedded)
    ├── index.html
    ├── assets/
    │   ├── index-[hash].js
    │   └── index-[hash].css
    └── vite.svg
```

## Configuration

The dashboard plugin currently doesn't require additional configuration. It uses embedded assets and serves them directly.

Future configuration options may include:

- Custom branding/theming
- Feature toggles
- Access control settings
- Analytics configuration

## Security

The dashboard plugin includes several security considerations:

- **Static Asset Serving**: Only serves pre-built, embedded assets
- **No Dynamic Content**: All content is static, reducing attack surface
- **Path Traversal Protection**: Uses Go's `fs.FS` interface for safe file access
- **Content Security Policy**: Implements CSP headers for XSS protection

### Access Control

Currently, the dashboard is publicly accessible. In production deployments, you should:

1. Implement authentication middleware
2. Add role-based access control (RBAC)
3. Use HTTPS in production
4. Configure proper CORS policies

## Testing

Run the plugin tests:

```bash
# Test the plugin
go test ./plugins/dashboard -v

# Test with coverage
go test ./plugins/dashboard -v -cover

# Run integration example
go run examples/dashboard/main.go
```

### Test Coverage

The plugin includes tests for:

- Plugin interface implementation
- Asset serving functionality
- Route registration
- Error handling

## Troubleshooting

### Common Issues

1. **Assets not loading**
   - Ensure the frontend was built: `cd frontend/dashboard && pnpm build`
   - Check that `dist/` directory exists and contains built files
   - Verify the embed directive is correct in `plugin.go`

2. **Routes not working**
   - Confirm the plugin is registered before `auth.Initialize()`
   - Check that AuthSome is mounted to your Forge app
   - Verify the base path configuration

3. **Development server issues**
   - Ensure Node.js 18+ is installed
   - Run `pnpm install` to install dependencies
   - Check for port conflicts (default: 5173)

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
// Enable debug logging (if available in your setup)
log.SetLevel(log.DebugLevel)
```

## Contributing

To contribute to the dashboard plugin:

1. **Frontend Changes**: Make changes in `frontend/dashboard/`
2. **Backend Changes**: Modify `plugins/dashboard/`
3. **Build**: Run `pnpm build` to update embedded assets
4. **Test**: Run tests and the integration example
5. **Submit**: Create a pull request with your changes

### Code Style

- **Go**: Follow standard Go conventions and gofmt
- **TypeScript**: Use ESLint and Prettier configurations
- **React**: Follow React best practices and hooks patterns
- **CSS**: Use Tailwind utility classes, avoid custom CSS

## License

This plugin is part of the AuthSome project and follows the same license terms.

## Support

For issues and questions:

- Check the [main AuthSome documentation](../../README.md)
- Review the [integration example](../../examples/dashboard/main.go)
- Open an issue in the AuthSome repository