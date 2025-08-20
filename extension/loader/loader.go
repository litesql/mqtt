package loader

import _ "github.com/litesql/mqtt/extension"

// #cgo CFLAGS: -DSQLITE_CORE
//
// #include "../../sqlite3.h"
//
// extern int sqlite3_extension_init(sqlite3*, char**, const sqlite3_api_routines);
import "C"

func init() {
	C.sqlite3_auto_extension((*[0]byte)(C.sqlite3_extension_init))
}
