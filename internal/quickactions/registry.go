// SPDX-License-Identifier: Apache-2.0

package quickactions

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/registry"
)

// RootKey returns the canonical registry key for a configured root index.
//
// Quick actions are stored in a single registry whose entries are namespaced
// by root: an entry's qualified name is "{rootKey}/{display-name}". This
// keeps lookups O(1) while still allowing several roots to expose actions
// that share a display name.
func RootKey(rootIndex int) string {
	return strconv.Itoa(rootIndex)
}

// BuildRegistry materializes the effective quick actions for every root in
// the given configuration and registers them all under their root-qualified
// keys. The returned registry is always non-nil; if no quick actions are
// declared, it is empty.
//
// Errors from individual entries (empty command, unknown subject, duplicate
// qualified name) are returned aggregated so the caller can surface them,
// but the rest of the registry is still built.
func BuildRegistry(cfg *config.Config) (*registry.Registry[*QuickAction], error) {
	if cfg == nil {
		return registry.New[*QuickAction](), nil
	}

	var (
		entries  []*QuickAction
		problems []string
		seen     = map[string]bool{}
	)

	collect := func(rootKey string, list []config.QuickActionConfig) {
		for _, entry := range list {
			qa, err := New(Params{
				RootKey:      rootKey,
				Subject:      entry.Subject,
				Name:         entry.Name,
				Description:  entry.Description,
				Command:      entry.Command,
				PreCommands:  entry.PreCommands,
				InitCommands: entry.InitCommands,
			})
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", rootKey, entry.Name, err))

				continue
			}

			if seen[qa.Name()] {
				problems = append(problems, qa.Name()+": duplicate")

				continue
			}

			seen[qa.Name()] = true
			entries = append(entries, qa)
		}
	}

	if len(cfg.Roots) == 0 {
		// With no roots configured, expose globals under a synthetic root key
		// so callers can still discover and run them (e.g. before the first
		// scan completes or when the wizard is opening for the first time).
		collect(RootKey(0), cfg.QuickActions)
	} else {
		for i := range cfg.Roots {
			collect(RootKey(i), cfg.QuickActionsForRoot(i))
		}
	}

	reg := registry.New(
		registry.With(slices.Values(entries)),
	)

	if len(problems) > 0 {
		return reg, fmt.Errorf("quickactions: %s", strings.Join(problems, "; "))
	}

	return reg, nil
}

// IterateForRoot returns an iterator over the quick actions registered for
// the given root index, optionally filtered by subject. Pass
// [models.SubjectNone] as the subject to receive every entry.
func IterateForRoot(
	reg *registry.Registry[*QuickAction],
	rootIndex int,
	subject models.SubjectKind,
) iter.Seq[*QuickAction] {
	prefix := RootKey(rootIndex) + "/"

	return func(yield func(*QuickAction) bool) {
		if reg == nil {
			return
		}

		for name, qa := range reg.All() {
			if !strings.HasPrefix(name, prefix) {
				continue
			}

			if subject != models.SubjectNone && qa.Subject() != subject {
				continue
			}

			if !yield(qa) {
				return
			}
		}
	}
}

// LookupForRoot resolves an action by its display name within the given
// root scope.
func LookupForRoot(
	reg *registry.Registry[*QuickAction],
	rootIndex int,
	displayName string,
) (*QuickAction, bool) {
	if reg == nil {
		return nil, false
	}

	return reg.Get(RootKey(rootIndex) + "/" + displayName)
}
