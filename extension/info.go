package extension

import (
	"fmt"

	"github.com/walterwanderley/sqlite"
)

var (
	version string
	commit  string
	date    string
)

type Info struct {
}

func (m *Info) Args() int {
	return 0
}

func (m *Info) Deterministic() bool {
	return true
}

func (m *Info) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	ctx.ResultText(fmt.Sprintf("github.com/litesql/mqtt version=%s, commit=%s, date=%s", version, commit, date))
}
