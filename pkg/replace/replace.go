package replace

import (
	"strings"
)

type ReplacerFunc func(string) (string, bool, error)

func Replace(s, startToken, endToken string, replacer ReplacerFunc) (string, error) {
	result := &strings.Builder{}
	for {
		before, tail, ok := strings.Cut(s, startToken)
		if !ok {
			result.WriteString(s)
			break
		}

		result.WriteString(before)

		expr, after, ok := strings.Cut(tail, endToken)
		if !ok {
			result.WriteString(startToken)
			s = tail
			continue
		}

		replaced, ok, err := replacer(expr)
		if err != nil {
			return "", err
		}
		if ok {
			result.WriteString(replaced)
		} else {
			result.WriteString(startToken)
			result.WriteString(expr)
			result.WriteString(endToken)
		}

		s = after
	}

	return result.String(), nil
}
