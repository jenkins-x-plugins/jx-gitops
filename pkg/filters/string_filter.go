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
}

// Matches returns true if the given filter matches the value
func (f *StringFilter) Matches(value string) bool {
	if f.Prefix != "" {
		if !HasPrefix(value, f.Prefix) {
			return false
		}
	}
	if f.Suffix != "" {
		if !HasSuffix(value, f.Suffix) {
			return false
		}
	}
	if f.Contains != "" {
		if !Contains(value, f.Contains) {
			return false
		}
	}
	return true
}

func HasPrefix(s, arg string) bool {
	if strings.HasPrefix(arg, "!") {
		return !HasPrefix(s, arg[1:])
	}
	return strings.HasPrefix(s, arg)
}

func HasSuffix(s, arg string) bool {
	if strings.HasPrefix(arg, "!") {
		return !HasSuffix(s, arg[1:])
	}
	return strings.HasSuffix(s, arg)
}

func Contains(s, arg string) bool {
	if strings.HasPrefix(arg, "!") {
		return !Contains(s, arg[1:])
	}
	return strings.Contains(s, arg)
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
func (f *StringFilter) AddFlags(cmd *cobra.Command, optionPrefix, message string) {
	cmd.Flags().StringVarP(&f.Prefix, optionPrefix+"-prefix", "", "", fmt.Sprintf("matches if %s has the given prefix", optionPrefix))
	cmd.Flags().StringVarP(&f.Suffix, optionPrefix+"-suffix", "", "", fmt.Sprintf("matches if %s has the given suffix", optionPrefix))
	cmd.Flags().StringVarP(&f.Contains, optionPrefix+"-contains", "", "", fmt.Sprintf("matches if %s contains the given text", optionPrefix))
}
