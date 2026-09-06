package model

type OrgItemDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StaffCount  int    `json:"staffCount"`
}

type CreateOrgItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateOrgItemRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type CopyOrgRequest struct {
	Kind            string   `json:"kind"`
	TargetClinicIDs []string `json:"targetClinicIds"`
}

type CopyOrgResult struct {
	ClinicID string `json:"clinicId"`
	Inserted int    `json:"inserted"`
}

type OrgListFilter struct {
	ClinicID string
	Page     int
	Limit    int
	Search   string
}
