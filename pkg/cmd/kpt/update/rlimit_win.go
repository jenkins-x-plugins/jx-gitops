//go:build windows

package update

// Sometimes kpt fails with Too many open files. This is an attempt to reduce the risk for that
// Would this be needed in windows?
func increaseRLimit() {
}
