package formconfig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidateSubmission validates a map of submitted field values against the
// form field definitions. Returns a map of field key to error message for
// any invalid fields. An empty map means all fields are valid.
func ValidateSubmission(fields []FormField, data map[string]string) map[string]string {
	errs := make(map[string]string)

	for _, f := range fields {
		val := strings.TrimSpace(data[f.Key])

		if f.Validation.Required && val == "" {
			errs[f.Key] = fmt.Sprintf("%s is required", f.Label)
			continue
		}

		if val == "" {
			continue
		}

		if f.Validation.MinLen > 0 && len(val) < f.Validation.MinLen {
			errs[f.Key] = fmt.Sprintf("%s must be at least %d characters", f.Label, f.Validation.MinLen)
			continue
		}

		if f.Validation.MaxLen > 0 && len(val) > f.Validation.MaxLen {
			errs[f.Key] = fmt.Sprintf("%s must be at most %d characters", f.Label, f.Validation.MaxLen)
			continue
		}

		if f.Validation.Pattern != "" {
			re, err := regexp.Compile(f.Validation.Pattern)
			if err == nil && !re.MatchString(val) {
				errs[f.Key] = fmt.Sprintf("%s has an invalid format", f.Label)
				continue
			}
		}

		if f.Type == FieldNumber {
			num, err := strconv.Atoi(val)
			if err != nil {
				errs[f.Key] = fmt.Sprintf("%s must be a number", f.Label)
				continue
			}

			if f.Validation.Min != nil && num < *f.Validation.Min {
				errs[f.Key] = fmt.Sprintf("%s must be at least %d", f.Label, *f.Validation.Min)
				continue
			}

			if f.Validation.Max != nil && num > *f.Validation.Max {
				errs[f.Key] = fmt.Sprintf("%s must be at most %d", f.Label, *f.Validation.Max)
				continue
			}
		}

		if (f.Type == FieldSelect || f.Type == FieldRadio) && len(f.Options) > 0 {
			valid := false
			for _, opt := range f.Options {
				if opt.Value == val {
					valid = true
					break
				}
			}

			if !valid {
				errs[f.Key] = fmt.Sprintf("%s has an invalid selection", f.Label)
			}
		}
	}

	return errs
}
