package model

type CopyCatalogueRequest struct {
	SourceClinicID  string   `json:"sourceClinicId"`
	TargetClinicIDs []string `json:"targetClinicIds"`
}

type CopyCatalogueResult struct {
	ClinicID            string `json:"clinicId"`
	Drugs               int    `json:"drugs"`
	Treatments          int    `json:"treatments"`
	RadiologyTests      int    `json:"radiologyTests"`
	Categories          int    `json:"categories"`
	Manufacturers       int    `json:"manufacturers"`
	Units               int    `json:"units"`
	RadiologyCategories int    `json:"radiologyCategories"`
}
