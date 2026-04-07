// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"encoding/json"
	"fmt"

	"github.com/fredbi/git-janitor/internal/models"
)

// marshalRepoInfo encodes a *models.RepoInfo to JSON bytes.
//
// Error fields (Err, FetchErr, Platform.Err) are zeroed before encoding
// because error is not JSON-serializable and these are transient state
// that doesn't belong in a cache.
func marshalRepoInfo(info *models.RepoInfo) ([]byte, error) {
	// Shallow-clone to avoid mutating the caller's data.
	clone := *info
	clone.Err = nil
	clone.FetchErr = nil

	if clone.Platform != nil {
		p := *clone.Platform
		p.Err = nil
		clone.Platform = &p
	}

	data, err := json.Marshal(&clone) //nolint:musttag // model types use default field names intentionally
	if err != nil {
		return nil, fmt.Errorf("codec: marshalling RepoInfo: %w", err)
	}

	return data, nil
}

// unmarshalRepoInfo decodes JSON bytes back to *models.RepoInfo.
func unmarshalRepoInfo(data []byte) (*models.RepoInfo, error) {
	var info models.RepoInfo
	if err := json.Unmarshal(data, &info); err != nil { //nolint:musttag // model types use default field names intentionally
		return nil, fmt.Errorf("codec: unmarshalling RepoInfo: %w", err)
	}

	return &info, nil
}

// marshalHistoryEntry encodes a models.HistoryEntry to JSON bytes.
func marshalHistoryEntry(entry models.HistoryEntry) ([]byte, error) {
	data, err := json.Marshal(entry) //nolint:musttag // model types use default field names intentionally
	if err != nil {
		return nil, fmt.Errorf("codec: marshalling HistoryEntry: %w", err)
	}

	return data, nil
}

// unmarshalHistoryEntry decodes JSON bytes back to models.HistoryEntry.
func unmarshalHistoryEntry(data []byte) (models.HistoryEntry, error) {
	var entry models.HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil { //nolint:musttag // model types use default field names intentionally
		return entry, fmt.Errorf("codec: unmarshalling HistoryEntry: %w", err)
	}

	return entry, nil
}
