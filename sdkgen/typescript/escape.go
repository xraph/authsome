package typescript

import "strings"

// jsDocText makes s safe inside a /** ... */ block: "*/" would close it, and
// a line break would start a line without the leading " * ".
var jsDocText = strings.NewReplacer(
	"*/", `*\/`,
	"\r\n", " ",
	"\n", " ",
	"\r", " ",
).Replace
