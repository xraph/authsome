package dart

import "strings"

// dartStringBody makes s safe between the single quotes of a Dart string
// literal: backslashes, quotes and dollars (interpolation) are escaped, and
// line breaks become spaces since the literal is single-line.
var dartStringBody = strings.NewReplacer(
	`\`, `\\`,
	`'`, `\'`,
	`$`, `\$`,
	"\r\n", " ",
	"\n", " ",
	"\r", " ",
).Replace
