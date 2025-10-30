# AuthSome CLI Tool

The AuthSome CLI tool provides comprehensive management capabilities for the AuthSome authentication framework. It includes commands for database migrations, code generation, data seeding, organization management, user management, and configuration management.

## Installation

Build the CLI tool from source:

```bash
go build -o authsome ./cmd/authsome
```

Or install it directly:

```bash
go install github.com/xraph/authsome/cmd/authsome@latest
```

## Global Flags

- `--config string`: Config file (default is $HOME/.authsome.yaml)
- `--verbose, -v`: Verbose output
- `--help, -h`: Help for any command
- `--version`: Show version information

## Commands Overview

### Database Migrations

Manage database schema migrations:

```bash
# Run pending migrations
authsome migrate up

# Rollback the last migration
authsome migrate down

# Show migration status
authsome migrate status

# Reset database (drop all tables and re-run migrations)
authsome migrate reset --confirm
```

### Code Generation

Generate keys, configurations, and boilerplate code:

```bash
# Generate RSA key pair for JWT/OIDC
authsome generate keys --output ./keys

# Generate sample configuration file
authsome generate config --mode standalone --output authsome.yaml
authsome generate config --mode saas --output authsome-saas.yaml

# Generate cryptographically secure secret
authsome generate secret
authsome generate secret --length 64
```

### Database Seeding

Seed the database with test data for development:

```bash
# Seed basic test data (organizations, users, roles)
authsome seed basic

# Seed test users
authsome seed users --count 50 --org org_id_here

# Seed test organizations
authsome seed orgs --count 10

# Clear all seeded data
authsome seed clear --confirm
```

### Organization Management

Manage organizations in SaaS mode:

```bash
# List all organizations
authsome org list

# Create a new organization
authsome org create --name "Acme Corp" --slug "acme"

# Show organization details
authsome org show org_id_here

# Delete an organization
authsome org delete org_id_here --confirm

# List organization members
authsome org members org_id_here

# Add member to organization
authsome org add-member org_id_here user_id_here --role admin

# Remove member from organization
authsome org remove-member org_id_here user_id_here
```

### User Management

Manage users across organizations:

```bash
# List all users
authsome user list

# List users in specific organization
authsome user list --org org_id_here

# Create a new user
authsome user create --email user@example.com --name "John Doe" --password password123

# Show user details
authsome user show user_id_here

# Delete a user
authsome user delete user_id_here --confirm

# Update user password
authsome user password user_id_here --password newpassword123

# Verify user email
authsome user verify user_id_here
```

### Configuration Management

Manage and validate configuration files:

```bash
# Validate configuration file
authsome config validate authsome.yaml

# Show current configuration
authsome config show authsome.yaml

# Initialize new configuration
authsome config init --mode standalone --output authsome.yaml
authsome config init --mode saas --output authsome-saas.yaml
```

## Configuration File Examples

### Standalone Mode

```yaml
# AuthSome Standalone Configuration
mode: standalone

database:
  url: "authsome.db"

server:
  host: "localhost"
  port: 8080
  cors:
    enabled: true
    origins: ["http://localhost:3000"]

session:
  secret: "your-session-secret-here"
  maxAge: 86400
  secure: false
  sameSite: "lax"

plugins:
  username:
    enabled: true
  twofa:
    enabled: true
    issuer: "YourApp"
```

### SaaS Mode

```yaml
# AuthSome SaaS Configuration
mode: saas

database:
  url: "postgres://user:password@localhost/authsome?sslmode=disable"

server:
  host: "0.0.0.0"
  port: 8080

session:
  secret: "your-session-secret-here"
  maxAge: 86400
  secure: true
  sameSite: "strict"

organizations:
  enabled: true
  allowCreation: true

plugins:
  username:
    enabled: true
  oauth:
    enabled: true
    providers:
      google:
        clientId: "your-google-client-id"
        clientSecret: "your-google-client-secret"
```

## Common Workflows

### Development Setup

1. Generate configuration:
   ```bash
   authsome generate config --mode standalone --output authsome.yaml
   ```

2. Run migrations:
   ```bash
   authsome migrate up
   ```

3. Seed test data:
   ```bash
   authsome seed basic
   ```

### Production Deployment

1. Generate secure secrets:
   ```bash
   authsome generate secret --length 64
   authsome generate keys --output ./keys
   ```

2. Validate configuration:
   ```bash
   authsome config validate --file authsome.yaml
   ```

3. Run migrations:
   ```bash
   authsome migrate up
   ```

### User Management

1. Create admin user:
   ```bash
   authsome user create --email admin@company.com --name "Admin User" --password securepassword
   ```

2. Create organization:
   ```bash
   authsome org create --name "My Company" --slug "mycompany"
   ```

3. Add user to organization:
   ```bash
   authsome org add-member org_id user_id --role admin
   ```

## Database Support

The CLI tool supports multiple database backends:

- **SQLite**: Default for development (`authsome.db`)
- **PostgreSQL**: Production recommended (`postgres://...`)
- **MySQL**: Alternative option (`mysql://...`)

## Security Considerations

1. **Secrets**: Always use strong, randomly generated secrets in production
2. **Database**: Use proper database credentials and SSL connections
3. **HTTPS**: Enable secure cookies and HTTPS in production
4. **Rate Limiting**: Configure appropriate rate limits for your use case

## Troubleshooting

### Common Issues

1. **Database connection errors**: Check database URL and credentials
2. **Migration failures**: Ensure database is accessible and has proper permissions
3. **Permission errors**: Run with appropriate user permissions for file operations

### Debug Mode

Use the `--verbose` flag for detailed output:

```bash
authsome --verbose migrate up
```

## Environment Variables

The CLI tool respects these environment variables:

- `AUTHSOME_CONFIG`: Path to configuration file
- `AUTHSOME_DATABASE_URL`: Database connection string
- `AUTHSOME_LOG_LEVEL`: Logging level (debug, info, warn, error)

## Contributing

When adding new CLI commands:

1. Create command file in `cmd/authsome/`
2. Follow the existing patterns for cobra commands
3. Add comprehensive help text and examples
4. Update this documentation

## License

This CLI tool is part of the AuthSome project and follows the same license terms.