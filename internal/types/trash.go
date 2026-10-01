package types

// TrashItem describes one trashed project as listed from `.trash`.
//
// Trash names on disk are `<project-id>-<unix-timestamp>`, which no human or
// agent can use directly. The listing carries the fields needed to pick an
// entry (number, title) alongside the exact trash name the restore call
// needs, so `trash list` output is actionable without a second lookup.
type TrashItem struct {
	TrashName string    `json:"trash_name" yaml:"trash_name"`
	ProjectID string    `json:"project_id" yaml:"project_id"`
	Number    int       `json:"number" yaml:"number"`
	Title     string    `json:"title" yaml:"title"`
	DeletedAt Timestamp `json:"deleted_at" yaml:"deleted_at"`
}
