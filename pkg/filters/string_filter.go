package filters

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// StringFilter allows a prefix/postfix/regex to be specified
type StringFilter struct {
	Prefix   string
	Suffix   string
	Contains string
	// TODO
	// Regex    string
	// Includes []string
	// Excludes []string
}

// Matches returns true if the given filter matches the value
func (f *StringFilter) Matches(value string) bool {
	if f.Prefix != "" {
		if !strings.HasPrefix(value, f.Prefix) {
			return false
		}
	}
	if f.Suffix != "" {
		if !strings.HasSuffix(value, f.Suffix) {
			return false
		}
	}
	if f.Contains != "" {
		if !strings.Contains(value, f.Contains) {
			return false
		}
	}
	return true
}

func (f *StringFilter) String() string {
	b := strings.Builder{}
	fn := func(name, value string) {
		if value != "" {
			if b.Len() > 0 {
				b.WriteString(" && ")
			}
			b.WriteString(fmt.Sprintf("%s = '%s'", name, value))
		}
	}
	fn("prefix", f.Prefix)
	fn("suffix", f.Suffix)
	fn("contains", f.Contains)
	return b.String()
}

// AddFlags adds cmd flags
func (f *StringFilter) AddFlags(cmd *cobra.Command, optionPrefix string, message string) {
	cmd.Flags().StringVarP(&f.Prefix, optionPrefix+"-prefix", "", "", "matches if %s has the given prefix")
	cmd.Flags().StringVarP(&f.Suffix, optionPrefix+"-suffix", "", "", "matches if %s has the given suffix")
	cmd.Flags().StringVarP(&f.Contains, optionPrefix+"-contains", "", "", "matches if %s contains the given text")
}
