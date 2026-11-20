package valueobjects

// Visibility determina qué detalle de una ubicación puede mostrarse.
type Visibility string

// LocationSummary contiene los datos de ubicación que pueden aparecer en búsquedas y avisos.
type LocationSummary struct{ Name, Address, Municipality, Department, Country, Zone string }

const (
	Restricted Visibility = "restricted"
	Public     Visibility = "public"
)
