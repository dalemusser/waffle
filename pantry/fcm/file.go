// fcm/file.go
package fcm

import "os"

func init() {
	readFileImpl = os.ReadFile
}
