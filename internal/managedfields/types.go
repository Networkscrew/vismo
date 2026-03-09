package managedfields

import "time"

// FieldOwner represents a single manager entry from managedFields.
type FieldOwner struct {
	Manager    string
	Operation  string
	Time       time.Time
	APIVersion string
}

// FieldTrace holds the resolved ownership for a specific field path.
type FieldTrace struct {
	FieldPath  string
	FinalValue interface{}
	Owners     []FieldOwner // newest-first; empty if managedFields unavailable
	Note       string       // set when managedFields is absent or field not found
}
