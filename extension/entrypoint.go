//go:build !zig

package extension

import (
	"github.com/walterwanderley/sqlite"
)

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo linux LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #cgo windows LDFLAGS: -Wl,--allow-multiple-definition
//
// #include "../sqlite3.h"
//
// extern int sqlite3_extension_init(sqlite3*, char**, const sqlite3_api_routines);
import "C"

func init() {
	sqlite.Register(registerFunc)
}
