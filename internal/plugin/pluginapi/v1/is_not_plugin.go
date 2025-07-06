//go:build !fabricplugin

package pluginapiv1

import (
	"github.com/hashicorp/go-hclog"
)

func loggerForGoplugin() hclog.Logger {
	panic("Attempted to run a plugin built without `fabricplugin` tag")
}
