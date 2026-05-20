package model

type UpdateInfo struct {
	Current      string `json:"current"`
	Latest       string `json:"latest"`
	Available    bool   `json:"available"`
	ReleaseNotes string `json:"release_notes"`
	PublishedAt  string `json:"published_at"`
	ReleaseURL   string `json:"release_url"`
}
