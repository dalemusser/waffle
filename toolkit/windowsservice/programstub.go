// toolkit/windowsservice/programstub.go
//go:build !windows

package windowsservice

import "errors"

// On non-Windows platforms, this package is essentially unavailable.
// You can still compile code that references the type, but you can't
// actually run as a Windows service here.

var ErrNotWindows = errors.New("windowsservice: not supported on this platform")
