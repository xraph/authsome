package language

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// BuiltinFunctions provides runtime implementations of custom functions
type BuiltinFunctions struct {
	context map[string]interface{}
}

// NewBuiltinFunctions creates a new builtin functions handler
func NewBuiltinFunctions(ctx map[string]interface{}) *BuiltinFunctions {
	return &BuiltinFunctions{context: ctx}
}

// HasRole checks if principal has a specific role
func (b *BuiltinFunctions) HasRole(role string) bool {
	principal, ok := b.context["principal"].(map[string]interface{})
	if !ok {
		return false
	}
	
	roles, ok := principal["roles"].([]interface{})
	if !ok {
		// Try string slice
		if rolesStr, ok := principal["roles"].([]string); ok {
			for _, r := range rolesStr {
				if strings.EqualFold(r, role) {
					return true
				}
			}
		}
		return false
	}
	
	for _, r := range roles {
		if roleStr, ok := r.(string); ok {
			if strings.EqualFold(roleStr, role) {
				return true
			}
		}
	}
	
	return false
}

// HasAnyRole checks if principal has any of the specified roles
func (b *BuiltinFunctions) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if b.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if principal has all specified roles
func (b *BuiltinFunctions) HasAllRoles(roles []string) bool {
	for _, role := range roles {
		if !b.HasRole(role) {
			return false
		}
	}
	return true
}

// InTimeRange checks if current time is within specified range (UTC, 24h format)
func (b *BuiltinFunctions) InTimeRange(start, end string) bool {
	request, ok := b.context["request"].(map[string]interface{})
	if !ok {
		return false
	}
	
	// Get current time from request or use now
	var currentTime time.Time
	if reqTime, ok := request["time"].(time.Time); ok {
		currentTime = reqTime
	} else {
		currentTime = time.Now().UTC()
	}
	
	// Parse start time
	startParts := strings.Split(start, ":")
	if len(startParts) != 2 {
		return false
	}
	startHour, err := strconv.Atoi(startParts[0])
	if err != nil {
		return false
	}
	startMin, err := strconv.Atoi(startParts[1])
	if err != nil {
		return false
	}
	
	// Parse end time
	endParts := strings.Split(end, ":")
	if len(endParts) != 2 {
		return false
	}
	endHour, err := strconv.Atoi(endParts[0])
	if err != nil {
		return false
	}
	endMin, err := strconv.Atoi(endParts[1])
	if err != nil {
		return false
	}
	
	// Convert current time to minutes since midnight
	currentMinutes := currentTime.Hour()*60 + currentTime.Minute()
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin
	
	// Handle ranges that cross midnight
	if endMinutes < startMinutes {
		return currentMinutes >= startMinutes || currentMinutes <= endMinutes
	}
	
	return currentMinutes >= startMinutes && currentMinutes <= endMinutes
}

// IsWeekday checks if current day is Monday-Friday
func (b *BuiltinFunctions) IsWeekday() bool {
	request, ok := b.context["request"].(map[string]interface{})
	if !ok {
		return false
	}
	
	var currentTime time.Time
	if reqTime, ok := request["time"].(time.Time); ok {
		currentTime = reqTime
	} else {
		currentTime = time.Now().UTC()
	}
	
	weekday := currentTime.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// IPInRange checks if request IP is in any of the specified CIDR ranges
func (b *BuiltinFunctions) IPInRange(cidrs []string) bool {
	request, ok := b.context["request"].(map[string]interface{})
	if !ok {
		return false
	}
	
	ipStr, ok := request["ip"].(string)
	if !ok {
		return false
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		
		if network.Contains(ip) {
			return true
		}
	}
	
	return false
}

// ResourceMatches checks if resource ID matches a wildcard pattern
func (b *BuiltinFunctions) ResourceMatches(pattern string) bool {
	resource, ok := b.context["resource"].(map[string]interface{})
	if !ok {
		return false
	}
	
	resourceID, ok := resource["id"].(string)
	if !ok {
		return false
	}
	
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	
	// Suffix wildcard: "project:*"
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(resourceID, prefix+":")
	}
	
	// Exact match
	return resourceID == pattern
}

// DaysSince calculates days since a timestamp
func (b *BuiltinFunctions) DaysSince(timestamp time.Time) int64 {
	request, ok := b.context["request"].(map[string]interface{})
	if !ok {
		return 0
	}
	
	var currentTime time.Time
	if reqTime, ok := request["time"].(time.Time); ok {
		currentTime = reqTime
	} else {
		currentTime = time.Now().UTC()
	}
	
	duration := currentTime.Sub(timestamp)
	return int64(duration.Hours() / 24)
}

// HoursSince calculates hours since a timestamp
func (b *BuiltinFunctions) HoursSince(timestamp time.Time) int64 {
	request, ok := b.context["request"].(map[string]interface{})
	if !ok {
		return 0
	}
	
	var currentTime time.Time
	if reqTime, ok := request["time"].(time.Time); ok {
		currentTime = reqTime
	} else {
		currentTime = time.Now().UTC()
	}
	
	duration := currentTime.Sub(timestamp)
	return int64(duration.Hours())
}

// InOrg checks if resource belongs to an organization
func (b *BuiltinFunctions) InOrg(orgID string) bool {
	resource, ok := b.context["resource"].(map[string]interface{})
	if !ok {
		return false
	}
	
	resourceOrgID, ok := resource["org_id"].(string)
	if !ok {
		return false
	}
	
	return resourceOrgID == orgID
}

// IsMemberOf checks if principal is member of an organization
func (b *BuiltinFunctions) IsMemberOf(orgID string) bool {
	principal, ok := b.context["principal"].(map[string]interface{})
	if !ok {
		return false
	}
	
	// Check direct org_id
	if principalOrgID, ok := principal["org_id"].(string); ok {
		if principalOrgID == orgID {
			return true
		}
	}
	
	// Check organizations array
	orgs, ok := principal["organizations"].([]interface{})
	if !ok {
		return false
	}
	
	for _, org := range orgs {
		if orgStr, ok := org.(string); ok {
			if orgStr == orgID {
				return true
			}
		}
	}
	
	return false
}

// CreateFunctionBindings creates function bindings for CEL evaluation
func CreateFunctionBindings(ctx map[string]interface{}) map[string]interface{} {
	builtins := NewBuiltinFunctions(ctx)
	
	return map[string]interface{}{
		"has_role": func(role string) bool {
			return builtins.HasRole(role)
		},
		"has_any_role": func(roles []string) bool {
			return builtins.HasAnyRole(roles)
		},
		"has_all_roles": func(roles []string) bool {
			return builtins.HasAllRoles(roles)
		},
		"in_time_range": func(start, end string) bool {
			return builtins.InTimeRange(start, end)
		},
		"is_weekday": func() bool {
			return builtins.IsWeekday()
		},
		"ip_in_range": func(cidrs []string) bool {
			return builtins.IPInRange(cidrs)
		},
		"resource_matches": func(pattern string) bool {
			return builtins.ResourceMatches(pattern)
		},
		"days_since": func(timestamp time.Time) int64 {
			return builtins.DaysSince(timestamp)
		},
		"hours_since": func(timestamp time.Time) int64 {
			return builtins.HoursSince(timestamp)
		},
		"in_org": func(orgID string) bool {
			return builtins.InOrg(orgID)
		},
		"is_member_of": func(orgID string) bool {
			return builtins.IsMemberOf(orgID)
		},
	}
}

// ValidateBuiltinFunctionCall validates a builtin function call
func ValidateBuiltinFunctionCall(name string, args []interface{}) error {
	switch name {
	case "has_role":
		if len(args) != 1 {
			return fmt.Errorf("has_role requires 1 argument, got %d", len(args))
		}
		if _, ok := args[0].(string); !ok {
			return fmt.Errorf("has_role requires string argument")
		}
	case "in_time_range":
		if len(args) != 2 {
			return fmt.Errorf("in_time_range requires 2 arguments, got %d", len(args))
		}
		if _, ok := args[0].(string); !ok {
			return fmt.Errorf("in_time_range start must be string")
		}
		if _, ok := args[1].(string); !ok {
			return fmt.Errorf("in_time_range end must be string")
		}
	case "ip_in_range":
		if len(args) != 1 {
			return fmt.Errorf("ip_in_range requires 1 argument, got %d", len(args))
		}
		// Validate it's a list
		if _, ok := args[0].([]interface{}); !ok {
			if _, ok := args[0].([]string); !ok {
				return fmt.Errorf("ip_in_range requires array argument")
			}
		}
	}
	return nil
}
