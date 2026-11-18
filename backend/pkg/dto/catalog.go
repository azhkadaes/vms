package dto

type CatalogOfficer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	VIP         bool   `json:"vip"`
	Active      bool   `json:"active"`
	JobTitle    string `json:"jobTitle,omitempty"`
	Directorate string `json:"directorate,omitempty"`
	Deputy      string `json:"deputy,omitempty"`
}
type CatalogRoom struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
type PublishedNotice struct {
	Version string `json:"version"`
	ID      string `json:"id"`
	EN      string `json:"en"`
}
type PublicCatalog struct {
	Officers           []CatalogOfficer `json:"officers"`
	Rooms              []CatalogRoom    `json:"rooms"`
	AttachmentsEnabled bool             `json:"attachmentsEnabled"`
	PrivacyNotice      *PublishedNotice `json:"privacyNotice,omitempty"`
}
