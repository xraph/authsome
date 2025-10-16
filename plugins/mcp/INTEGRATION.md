# MCP Plugin - Integration Guide

Quick guide to integrate and test the MCP plugin with AuthSome.

## Step 1: Update Your Config

Add MCP plugin configuration to your YAML config:

```yaml
# authsome-standalone.yaml or authsome-proper.yaml
plugins:
  mcp:
    enabled: true
    mode: development  # or readonly, admin
    transport: stdio
    authorization:
      require_api_key: false  # Set true in production
      allowed_operations:
        - query_user
        - check_permission
        - search_audit_logs
    rate_limit:
      requests_per_minute: 60
```

## Step 2: Test with CLI

```bash
# Start MCP server
go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml

# In another terminal, test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml
```

Expected output:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "0.1.0",
    "serverInfo": {
      "name": "authsome-mcp",
      "version": "0.1.0"
    }
  }
}
```

## Step 3: Test Resources

```bash
# List available resources
echo '{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml

# Read config resource
echo '{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"authsome://config"}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml

# Read schema resource
echo '{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"authsome://schema"}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml
```

## Step 4: Test Tools

```bash
# List available tools
echo '{"jsonrpc":"2.0","id":5,"method":"tools/list","params":{}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml

# Query user by email (after creating a user)
echo '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"query_user","arguments":{"email":"test@example.com"}}}' | \
  go run cmd/authsome-cli/main.go mcp serve --config=authsome-standalone.yaml
```

## Step 5: Run Tests

```bash
# Run MCP plugin tests
go test ./plugins/mcp/... -v

# With coverage
go test ./plugins/mcp/... -coverprofile=mcp-coverage.out
go tool cover -html=mcp-coverage.out
```

## Step 6: Build CLI with MCP

```bash
# Build CLI binary
go build -o authsome-cli ./cmd/authsome-cli/

# Run MCP server
./authsome-cli mcp serve --config=authsome-standalone.yaml

# Show help
./authsome-cli mcp --help
./authsome-cli mcp serve --help
```

## Integration with AI Assistants

### Claude Desktop

1. Install Claude Desktop from Anthropic
2. Configure MCP server in Claude's config:

```json
{
  "mcpServers": {
    "authsome": {
      "command": "/path/to/authsome-cli",
      "args": ["mcp", "serve", "--config=/path/to/config.yaml"],
      "env": {}
    }
  }
}
```

3. Restart Claude Desktop
4. Ask Claude: "Show me the user with email test@example.com"

### Custom Integration

```go
package main

import (
    "context"
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/mcp"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New(
        authsome.WithDatabase(db),
        authsome.WithForgeConfig(cfg),
    )
    
    // Create MCP plugin
    mcpPlugin := mcp.NewPlugin(mcp.Config{
        Enabled: true,
        Mode: mcp.ModeReadOnly,
        Transport: mcp.TransportStdio,
    })
    
    // Register and initialize
    auth.RegisterPlugin(mcpPlugin)
    auth.Initialize(context.Background())
    
    // Start MCP server
    mcpPlugin.Start(context.Background())
}
```

## Troubleshooting

### "database not available"
- Ensure `--config` points to valid config file with database connection
- Check database is running and accessible

### "user not found"
- Create test user first: `authsome-cli user create ...`
- Or use `--mode=development` to create test data via MCP

### "operation not allowed"
- Check operation is in `allowed_operations` list
- Or change mode to `admin` or `development`

### No response from server
- Check if server started successfully
- Look for error messages in stderr
- Verify JSON-RPC request format

## Security Checklist

- [ ] Use `mode: readonly` in production
- [ ] Enable `require_api_key: true` in production
- [ ] Set appropriate rate limits
- [ ] Monitor audit logs for MCP operations
- [ ] Never use `development` mode in production
- [ ] Use HTTPS for HTTP transport
- [ ] Validate API keys against database
- [ ] Implement key rotation policy

## Next Steps

1. **Test thoroughly** - Run through all examples
2. **Integrate with CI/CD** - Add MCP tests to pipeline
3. **Connect AI assistant** - Configure Claude Desktop or similar
4. **Monitor usage** - Check audit logs for MCP operations
5. **Gather feedback** - See what resources/tools users need most
6. **Iterate** - Add more resources and tools based on usage

## Resources

- Full documentation: `plugins/mcp/README.md`
- Design decisions: `plugins/mcp/DESIGN.md`
- Implementation summary: `MCP_PLUGIN_SUMMARY.md`
- Tests: `plugins/mcp/plugin_test.go`

