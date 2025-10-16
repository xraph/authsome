# AuthSome CLI Tool

The AuthSome CLI tool provides comprehensive management capabilities for the AuthSome authentication framework. It includes commands for database migrations, code generation, data seeding, organization management, user management, and configuration management.

## Installation

Build the CLI tool from source:

```bash
go build -o authsome-cli ./cmd/authsome-cli
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
./authsome-cli migrate up

# Rollback the last migration
./authsome-cli migrate down

# Show migration status
./authsome-cli migrate status

# Reset database (drop all tables and re-run migrations)
./authsome-cli migrate reset --confirm
```

### Code Generation

Generate keys, configurations, and boilerplate code:

```bash
# Generate RSA key pair for JWT/OIDC
./authsome-cli generate keys --output ./keys

# Generate sample configuration file
./authsome-cli generate config --mode standalone --output authsome.yaml
./authsome-cli generate config --mode saas --output authsome-saas.yaml

# Generate cryptographically secure secret
./authsome-cli generate secret
./authsome-cli generate secret --length 64
```

### Database Seeding

Seed the database with test data for development:

```bash
# Seed basic test data (organizations, users, roles)
./authsome-cli seed basic

# Seed test users
./authsome-cli seed users --count 50 --org org_id_here

# Seed test organizations
./authsome-cli seed orgs --count 10

# Clear all seeded data
./authsome-cli seed clear --confirm
```

### Organization Management

Manage organizations in SaaS mode:

```bash
# List all organizations
./authsome-cli org list

# Create a new organization
./authsome-cli org create --name "Acme Corp" --slug "acme"

# Show organization details
./authsome-cli org show org_id_here

# Delete an organization
./authsome-cli org delete org_id_here --confirm

# List organization members
./authsome-cli org members org_id_here

# Add member to organization
./authsome-cli org add-member org_id_here user_id_here --role admin

# Remove member from organization
./authsome-cli org remove-member org_id_here user_id_here
```

### User Management

Manage users across organizations:

```bash
# List all users
./authsome-cli user list

# List users in specific organization
./authsome-cli user list --org org_id_here

# Create a new user
./authsome-cli user create --email user@example.com --name "John Doe" --password password123

# Show user details
./authsome-cli user show user_id_here

# Delete a user
./authsome-cli user delete user_id_here --confirm

# Update user password
./authsome-cli user password user_id_here --password newpassword123

# Verify user email
./authsome-cli user verify user_id_here
```

### Configuration Management

Manage and validate configuration files:

```bash
# Validate configuration file
./authsome-cli config validate authsome.yaml

# Show current configuration
./authsome-cli config show authsome.yaml

# Initialize new configuration
./authsome-cli config init --mode standalone --output authsome.yaml
./authsome-cli config init --mode saas --output authsome-saas.yaml
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
   ./authsome-cli generate config --mode standalone --output authsome.yaml
   ```

2. Run migrations:
   ```bash
   ./authsome-cli migrate up
   ```

3. Seed test data:
   ```bash
   ./authsome-cli seed basic
   ```

### Production Deployment

1. Generate secure secrets:
   ```bash
   ./authsome-cli generate secret --length 64
   ./authsome-cli generate keys --output ./keys
   ```

2. Validate configuration:
   ```bash
   ./authsome-cli config validate --file authsome.yaml
   ```

3. Run migrations:
   ```bash
   ./authsome-cli migrate up
   ```

### User Management

1. Create admin user:
   ```bash
   ./authsome-cli user create --email admin@company.com --name "Admin User" --password securepassword
   ```

2. Create organization:
   ```bash
   ./authsome-cli org create --name "My Company" --slug "mycompany"
   ```

3. Add user to organization:
   ```bash
   ./authsome-cli org add-member org_id user_id --role admin
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
./authsome-cli --verbose migrate up
```

## Environment Variables

The CLI tool respects these environment variables:

- `AUTHSOME_CONFIG`: Path to configuration file
- `AUTHSOME_DATABASE_URL`: Database connection string
- `AUTHSOME_LOG_LEVEL`: Logging level (debug, info, warn, error)

## Contributing

When adding new CLI commands:

1. Create command file in `cmd/authsome-cli/`
2. Follow the existing patterns for cobra commands
3. Add comprehensive help text and examples
4. Update this documentation

## License

This CLI tool is part of the AuthSome project and follows the same license terms.