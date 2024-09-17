package util

import (
	"fmt"
	"regexp"
	"strings"
)

func WildcardToRegexpStr(pattern string) string {
	formatRegexp := func(s string) string {
		return fmt.Sprintf("(?i)^%s$", s)
	}

	components := strings.Split(pattern, "*")
	if len(components) == 1 {
		// if len is 1, there are no *'s, return exact match pattern
		return formatRegexp(pattern)
	}
	var result strings.Builder
	for i, literal := range components {

		// Replace * with .*
		if i > 0 {
			result.WriteString(".*")
		}

		// Quote any regular expression meta characters in the
		// literal text.
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return formatRegexp(result.String())
}

func WildcardToRegexp(pattern string) *regexp.Regexp {
	return regexp.MustCompile(WildcardToRegexpStr(pattern))
}
