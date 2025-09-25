package terraform

import (
	"regexp"
	"strings"
)

type TokenMatch struct {
	Token    string
	Contents string
	Start    int
	End      int
}

// extractTokenContents extracts the contents between ${} from a token string
func extractTokenContents(token string) (string, bool) {
	if matches := tokenPattern.FindStringSubmatch(token); len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func findAllTokens(input string) []TokenMatch {
	matches := allTokensPattern.FindAllStringSubmatch(input, -1)
	indices := allTokensPattern.FindAllStringIndex(input, -1)

	if len(matches) != len(indices) {
		return nil
	}

	tokens := make([]TokenMatch, len(matches))
	for i, match := range matches {
		tokens[i] = TokenMatch{
			Token:    match[0],
			Contents: match[1],
			Start:    indices[i][0],
			End:      indices[i][1],
		}
	}

	return tokens
}

func isOnlyToken(input string) bool {
	return strings.TrimSpace(input) == strings.TrimSpace(allTokensPattern.FindString(input))
}

var tokenPattern = regexp.MustCompile(`^\${([^}]+)}$`)
var allTokensPattern = regexp.MustCompile(`\$\{([^}]+)\}`)
