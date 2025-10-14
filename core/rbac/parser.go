package rbac

import (
    "fmt"
    "strings"
)

// Parser parses RBAC policy expressions.
// Supported syntax:
//   "subject:action1,action2 on resource[:*] [where key = value]"
// Examples:
//   "user:read on organization:123"
//   "project:* on organization:*"           // actions omitted -> invalid
//   "invoice:read,write on organization:* where role = admin"
//   "user:create,read,update,delete on *"
type Parser struct{}

func NewParser() *Parser { return &Parser{} }

// Parse converts a policy expression string into a Policy struct.
func (p *Parser) Parse(expression string) (*Policy, error) {
    expr := strings.TrimSpace(expression)
    if expr == "" {
        return nil, fmt.Errorf("empty policy expression")
    }

    // split condition if present
    var condition string
    parts := strings.SplitN(expr, " where ", 2)
    head := parts[0]
    if len(parts) == 2 {
        condition = strings.TrimSpace(parts[1])
    }

    // split subject/actions and resource
    main := strings.SplitN(head, " on ", 2)
    if len(main) != 2 {
        return nil, fmt.Errorf("policy missing 'on' resource clause")
    }

    left := strings.TrimSpace(main[0]) // subject:actions
    resource := strings.TrimSpace(main[1])
    if left == "" || resource == "" {
        return nil, fmt.Errorf("policy missing subject/actions or resource")
    }

    // subject:actions (subject may contain ':' e.g., "role:owner")
    // Split by the LAST ':' to allow subjects with embedded colons.
    idx := strings.LastIndex(left, ":")
    if idx <= 0 || idx >= len(left)-1 {
        return nil, fmt.Errorf("policy left side must be 'subject:actions'")
    }
    subject := strings.TrimSpace(left[:idx])
    actionsStr := strings.TrimSpace(left[idx+1:])
    if subject == "" || actionsStr == "" {
        return nil, fmt.Errorf("policy requires subject and at least one action")
    }

    actionTokens := strings.Split(actionsStr, ",")
    actions := make([]string, 0, len(actionTokens))
    for _, a := range actionTokens {
        a = strings.TrimSpace(a)
        if a == "" {
            continue
        }
        actions = append(actions, a)
    }
    if len(actions) == 0 {
        return nil, fmt.Errorf("policy requires at least one action")
    }

    return &Policy{
        Subject:   subject,
        Actions:   actions,
        Resource:  resource,
        Condition: condition,
    }, nil
}