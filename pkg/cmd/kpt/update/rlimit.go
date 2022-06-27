//go:build !windows

package update

import (
	"syscall"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

// Sometimes kpt fails with Too many open files. This is an attempt to reduce the risk for that
func increaseRLimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Logger().Infof("Can't get file handle limit: %v", err)
	}
	if rLimit.Cur < 10240 {
		rLimit.Cur = 10240
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			log.Logger().Infof("Can't increase file handle limit: %v", err)
		}
	}
}
