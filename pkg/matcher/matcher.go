package matcher

import (
	"regexp"

	"github.com/pkg/errors"
)

// Matcher matches strings using regex
type Matcher struct {
	Includes []*regexp.Regexp
	Excludes []*regexp.Regexp
}

// Matches returns true if there are no includes or one of them matches and the text does not match an exclude
func (m *Matcher) Matches(text string) bool {
	matches := len(m.Includes) == 0
	for _, re := range m.Includes {
		if re.MatchString(text) {
			matches = true
			break
		}
	}
	if !matches {
		return false
	}
	for _, re := range m.Excludes {
		if re.MatchString(text) {
			return false
		}
	}
	return true
}

// ToRegexs creates a slice of regex
func (m *Matcher) ToRegexs(texts []string) ([]*regexp.Regexp, error) {
	var answer []*regexp.Regexp
	for _, text := range texts {
		re, err := regexp.Compile(text)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to parse regex: %s", text)
		}
		answer = append(answer, re)
	}
	return answer, nil
}
