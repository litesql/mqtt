//go:build zig

package extension

// #cgo CFLAGS: -DSQLITE_CORE
//
// #include "../sqlite3.h"
//
// extern int sqlite3_extension_init(sqlite3*, char**, const sqlite3_api_routines);
import "C"
import "github.com/walterwanderley/sqlite"

func init() {
	sqlite.Register(registerFunc)
}
