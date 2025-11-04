# Dashboard Components (Gomponents)

This directory contains the gomponents-based UI components for the minimal AuthSome dashboard plugin.

## Structure

```
components/
├── README.md              # This file
├── base.go                # Base HTML layout with head, body, navigation
├── navigation.go          # Header, nav menu, footer, icons
└── pages/                 # Individual page components
    ├── dashboard.go       # Dashboard stats & activity
    ├── error.go           # Generic error page
    ├── login.go           # Standalone login page
    ├── notfound.go        # 404 page
    └── users.go           # Users list with search/pagination
```

## Key Components

### Base Layout
```go
import "github.com/xraph/authsome/plugins/dashboard/components"

pageData := components.PageData{
    Title:      "Dashboard",
    User:       currentUser,
    ActivePage: "dashboard",
    CSRFToken:  token,
    BasePath:   "/auth",
    Year:       2025,
}

page := components.BaseLayout(pageData, yourContentHere)
page.Render(w)
```

### Pages
```go
import "github.com/xraph/authsome/plugins/dashboard/components/pages"

// Dashboard page
content := pages.DashboardPage(stats, basePath)

// Users list
content := pages.UsersPage(usersData, basePath)

// Login (standalone - no base layout)
page := pages.Login(loginData)

// 404 (standalone)
page := pages.NotFound(basePath)
```

## Icons

Uses [gomponents-lucide](https://github.com/eduardolat/gomponents-lucide):

```go
import "github.com/eduardolat/gomponents-lucide"

lucide.Users(Class("h-5 w-5"))
lucide.ShieldCheck(Class("h-6 w-6 text-violet-600"))
lucide.Search(Class("h-4 w-4"))
```

**Available icons used:**
- Users, UserCircle
- ShieldCheck, Lock, AlertCircle, CheckCircle
- Search, X, ArrowRight, ArrowLeft
- Home, Settings, Menu
- Sun, Moon (theme toggle)
- Heart, ChevronDown, TrendingUp
- And more...

## Adding a New Page

1. **Create `components/pages/mypage.go`:**
```go
package pages

import (
    "github.com/eduardolat/gomponents-lucide"
    g "maragu.dev/gomponents"
    . "maragu.dev/gomponents/html"
)

func MyPage(data MyData, basePath string) g.Node {
    return Div(
        Class("space-y-6"),
        H1(Class("text-2xl font-bold"), g.Text("My Page")),
        // ... your content
    )
}
```

2. **Use in handler:**
```go
func (h *Handler) ServeMyPage(c forge.Context) error {
    pageData := components.PageData{
        Title:      "My Page",
        ActivePage: "mypage",
        // ...
    }
    
    content := pages.MyPage(myData, h.basePath)
    page := components.BaseLayout(pageData, content)
    
    return h.render(c, page)
}
```

## Alpine.js Integration

For client-side interactivity (dropdowns, tabs, theme toggle):

```go
// Alpine.js directive as attributes
Button(
    g.Attr("@click", "toggleDropdown()"),
    g.Attr("x-show", "isOpen"),
    // ...
)

// x-data for component state
Div(
    g.Attr("x-data", "{ open: false }"),
    // ...
)

// Conditional rendering
Div(
    g.Attr("x-show", "activeTab === 'users'"),
    g.Attr("x-cloak", ""), // Hide until Alpine initializes
    // ...
)
```

## Styling

Uses **Tailwind CSS** via CDN (configured in `base.go`):

```go
// Responsive classes
Class("hidden lg:flex")

// Dark mode
Class("bg-white dark:bg-gray-900")

// States
Class("hover:bg-gray-50 active:bg-gray-100")

// Custom colors (configured in base.go)
Class("bg-violet-600 text-white")
```

## Testing

```go
func TestDashboardPage(t *testing.T) {
    stats := &pages.DashboardStats{
        TotalUsers: 100,
        UserGrowth: 10.5,
    }
    
    html := pages.DashboardPage(stats, "/auth").String()
    
    assert.Contains(t, html, "100")
    assert.Contains(t, html, "Total Users")
}
```

## Performance

- **Compile-time safety**: No runtime template errors
- **Fast rendering**: No template parsing overhead
- **Type-safe**: Full Go type checking
- **Cacheable**: Render once, cache as needed

## Migration from Templates

See `GOMPONENTS_MIGRATION_SUMMARY.md` in parent directory for:
- Migration status
- Remaining tasks
- Handler updates
- Testing strategy

## References

- [Gomponents](https://www.gomponents.com)
- [Lucide Icons](https://lucide.dev)
- [Tailwind CSS](https://tailwindcss.com)
- [Alpine.js](https://alpinejs.dev)

