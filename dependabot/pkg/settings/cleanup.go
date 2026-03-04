package settings

import "sync/atomic"

var cleanupTempDirsEnabled atomic.Bool

func init() {
	cleanupTempDirsEnabled.Store(true)
}

// SetCleanupTempDirsEnabled sets whether temporary directories should be cleaned up.
func SetCleanupTempDirsEnabled(enabled bool) {
	cleanupTempDirsEnabled.Store(enabled)
}

// CleanupTempDirsEnabled returns whether temporary directories should be cleaned up.
func CleanupTempDirsEnabled() bool {
	return cleanupTempDirsEnabled.Load()
}
