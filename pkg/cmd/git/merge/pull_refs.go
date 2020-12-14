package merge

import (
	"fmt"
	"strings"
)

// PullRefs is the result of parsing the Prow PULL_REFS
type PullRefs struct {
	BaseBranch string
	BaseSha    string
	ToMerge    []MergePair
}

type MergePair struct {
	Key string
	SHA string
}

// ParsePullRefs parses the Prow PULL_REFS env var formatted string and converts to a map of branch:sha
func ParsePullRefs(pullRefs string) (*PullRefs, error) {
	kvs := strings.Split(pullRefs, ",")
	answer := PullRefs{}
	for i, kv := range kvs {
		s := strings.Split(kv, ":")
		if len(s) != 2 {
			return nil, fmt.Errorf("incorrect format for branch:sha %s", kv)
		}
		if i == 0 {
			answer.BaseBranch = s[0]
			answer.BaseSha = s[1]
		} else {
			answer.ToMerge = append(answer.ToMerge, MergePair{Key: s[0], SHA: s[1]})
		}
	}
	return &answer, nil
}

func (pr *PullRefs) String() string {
	s := fmt.Sprintf("%s:%s", pr.BaseBranch, pr.BaseSha)
	for _, p := range pr.ToMerge {
		s = fmt.Sprintf("%s,%s:%s", s, p.Key, p.SHA)
	}
	return s
}
