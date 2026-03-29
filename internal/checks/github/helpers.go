// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
)

func singleAlert(a engine.Alert) iter.Seq[engine.Alert] {
	return func(yield func(engine.Alert) bool) {
		yield(a)
	}
}
