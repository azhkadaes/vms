package entity

// CatalogOfficer derives VIP from Jabatan; it carries no staff assignments.
type CatalogOfficer struct {
	ID          int32
	Name        string
	VIP         bool
	Active      bool
	JobTitle    string
	Directorate string
	Deputy      string
}

// CatalogRoom excludes private room metadata from the public destination list.
type CatalogRoom struct {
	ID     int32
	Name   string
	Active bool
}

// PublicCatalog holds the two destination lists from one consistent read snapshot.
type PublicCatalog struct {
	Officers []CatalogOfficer
	Rooms    []CatalogRoom
}
