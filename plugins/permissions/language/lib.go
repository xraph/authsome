package language

import (
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// authsomeLib implements cel.Library for AuthSome custom functions.
type authsomeLib struct{}

// LibraryName returns the name of this library.
func (authsomeLib) LibraryName() string {
	return "authsome.permissions"
}

// CompileOptions returns compilation options for custom functions.
func (authsomeLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		// Context-dependent functions (declared but not fully implemented yet)
		// These will be enhanced in Week 2 with proper context resolution
		cel.Function("has_role",
			cel.Overload("has_role_string",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.UnaryBinding(func(role ref.Val) ref.Val {
					// Placeholder - returns false
					// Real implementation in Week 2 will access principal.roles
					return types.Bool(false)
				}))),

		cel.Function("has_any_role",
			cel.Overload("has_any_role_list",
				[]*cel.Type{cel.ListType(cel.StringType)},
				cel.BoolType,
				cel.UnaryBinding(func(roles ref.Val) ref.Val {
					return types.Bool(false)
				}))),

		cel.Function("has_all_roles",
			cel.Overload("has_all_roles_list",
				[]*cel.Type{cel.ListType(cel.StringType)},
				cel.BoolType,
				cel.UnaryBinding(func(roles ref.Val) ref.Val {
					return types.Bool(false)
				}))),

		cel.Function("in_time_range",
			cel.Overload("in_time_range_strings",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(func(start, end ref.Val) ref.Val {
					// Placeholder implementation
					return types.Bool(false)
				}))),

		cel.Function("is_weekday",
			cel.Overload("is_weekday",
				[]*cel.Type{},
				cel.BoolType,
				cel.FunctionBinding(func(vals ...ref.Val) ref.Val {
					now := time.Now().UTC()
					weekday := now.Weekday()

					return types.Bool(weekday >= time.Monday && weekday <= time.Friday)
				}))),

		cel.Function("ip_in_range",
			cel.Overload("ip_in_range_list",
				[]*cel.Type{cel.ListType(cel.StringType)},
				cel.BoolType,
				cel.UnaryBinding(func(cidrs ref.Val) ref.Val {
					// Placeholder implementation
					return types.Bool(false)
				}))),

		cel.Function("resource_matches",
			cel.Overload("resource_matches_string",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.UnaryBinding(func(pattern ref.Val) ref.Val {
					// Placeholder implementation
					return types.Bool(false)
				}))),

		cel.Function("days_since",
			cel.Overload("days_since_timestamp",
				[]*cel.Type{cel.TimestampType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					timestamp, ok := arg.Value().(time.Time)
					if !ok {
						return types.NewErr("days_since requires timestamp argument")
					}

					duration := time.Since(timestamp)

					return types.Int(int64(duration.Hours() / 24))
				}))),

		cel.Function("hours_since",
			cel.Overload("hours_since_timestamp",
				[]*cel.Type{cel.TimestampType},
				cel.IntType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					timestamp, ok := arg.Value().(time.Time)
					if !ok {
						return types.NewErr("hours_since requires timestamp argument")
					}

					duration := time.Since(timestamp)

					return types.Int(int64(duration.Hours()))
				}))),

		cel.Function("in_org",
			cel.Overload("in_org_string",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.UnaryBinding(func(orgID ref.Val) ref.Val {
					// Placeholder implementation
					return types.Bool(false)
				}))),

		cel.Function("is_member_of",
			cel.Overload("is_member_of_string",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.UnaryBinding(func(groupID ref.Val) ref.Val {
					// Placeholder implementation
					return types.Bool(false)
				}))),
	}
}

// ProgramOptions returns runtime program options.
func (authsomeLib) ProgramOptions() []cel.ProgramOption {
	// No additional program options needed
	// Function bindings are already provided in CompileOptions
	return []cel.ProgramOption{}
}

// Helper function to check if a principal has a role (used by evaluator in Week 2+).
func HasRole(principal map[string]any, role string) bool {
	if principal == nil {
		return false
	}

	roles, ok := principal["roles"]
	if !ok {
		return false
	}

	// Handle []interface{}
	if roleList, ok := roles.([]any); ok {
		for _, r := range roleList {
			if roleStr, ok := r.(string); ok {
				if strings.EqualFold(roleStr, role) {
					return true
				}
			}
		}

		return false
	}

	// Handle []string
	if roleList, ok := roles.([]string); ok {
		for _, r := range roleList {
			if strings.EqualFold(r, role) {
				return true
			}
		}

		return false
	}

	return false
}
