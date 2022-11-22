package watchdog

const (
	ddeLockTaskName    = "dde-lock"
	ddeLockServiceName = "org.deepin.dde.LockFront1"
)

func launchDdeLock() error {
	return startService(ddeLockServiceName)
}

func newDdeLock(getLockedFn func() bool) *taskInfo {
	isDdeLockRunning := func() (bool, error) {
		if getLockedFn() {
			return isDBusServiceExist(ddeLockServiceName)
		} else {
			return false, errNoNeedLaunch
		}
	}
	return newTaskInfo(ddeLockTaskName, isDdeLockRunning, launchDdeLock)
}
