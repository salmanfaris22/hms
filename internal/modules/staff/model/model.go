package model

import "time"

type StaffDTO struct {
	ID                string    `json:"id"`
	StaffCode         string    `json:"staffCode"`
	Email             string    `json:"email"`
	FullName          string    `json:"fullName"`
	FirstName         string    `json:"firstName"`
	MiddleName        string    `json:"middleName"`
	LastName          string    `json:"lastName"`
	ProfilePhoto      string    `json:"profilePhoto"`
	DepartmentID      string    `json:"departmentId,omitempty"`
	DepartmentName    string    `json:"departmentName"`
	DesignationID     string    `json:"designationId,omitempty"`
	DesignationName   string    `json:"designationName"`
	SpecializationID  string    `json:"specializationId,omitempty"`
	SpecializationName string   `json:"specializationName"`
	MobileCountry     string    `json:"mobileCountry"`
	MobileNumber      string    `json:"mobileNumber"`
	AdditionalMobile  string    `json:"additionalMobile"`
	LandlineNumber    string    `json:"landlineNumber"`
	ViewInEMR         bool      `json:"viewInEmr"`
	JoinDate          time.Time `json:"joinDate"`
	Status            string    `json:"status"`
	IsActive          bool      `json:"isActive"`
	WorkLocations     []string  `json:"workLocations"`
}

type ListFilter struct {
	ClinicID     string
	Page         int
	Limit        int
	Search       string
	DepartmentID string
	Status       string
}

type CreateStaffRequest struct {
	StaffCode         string   `json:"staffCode"`
	AutoGenerateCode  bool     `json:"autoGenerateCode"`
	Email             string   `json:"email"`
	Password          string   `json:"password"`
	FirstName         string   `json:"firstName"`
	MiddleName        string   `json:"middleName"`
	LastName          string   `json:"lastName"`
	ProfilePhoto      string   `json:"profilePhoto"`
	DepartmentID      string   `json:"departmentId"`
	DesignationID     string   `json:"designationId"`
	SpecializationID  string   `json:"specializationId"`
	MobileCountry     string   `json:"mobileCountry"`
	MobileNumber      string   `json:"mobileNumber"`
	AdditionalMobile  string   `json:"additionalMobile"`
	LandlineNumber    string   `json:"landlineNumber"`
	ViewInEMR         bool     `json:"viewInEmr"`
	WorkLocations     []string `json:"workLocations"`
}

type UpdateStaffRequest struct {
	FirstName         *string   `json:"firstName,omitempty"`
	MiddleName        *string   `json:"middleName,omitempty"`
	LastName          *string   `json:"lastName,omitempty"`
	ProfilePhoto      *string   `json:"profilePhoto,omitempty"`
	DepartmentID      *string   `json:"departmentId,omitempty"`
	DesignationID     *string   `json:"designationId,omitempty"`
	SpecializationID  *string   `json:"specializationId,omitempty"`
	MobileCountry     *string   `json:"mobileCountry,omitempty"`
	MobileNumber      *string   `json:"mobileNumber,omitempty"`
	AdditionalMobile  *string   `json:"additionalMobile,omitempty"`
	LandlineNumber    *string   `json:"landlineNumber,omitempty"`
	ViewInEMR         *bool     `json:"viewInEmr,omitempty"`
	Status            *string   `json:"status,omitempty"`
	WorkLocations     *[]string `json:"workLocations,omitempty"`
}

type StatsDTO struct {
	TotalStaff    int `json:"totalStaff"`
	ActiveToday   int `json:"activeToday"`
	OnLeave       int `json:"onLeave"`
	Departments   int `json:"departments"`
}

// ── Roles ───────────────────────────────────────────────────────────────────

type RoleDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StaffCount  int       `json:"staffCount"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Permissions *[]string `json:"permissions,omitempty"`
}

type AssignRolesRequest struct {
	UserIDs []string `json:"userIds"`
	RoleIDs []string `json:"roleIds"`
}

// ── Documents ───────────────────────────────────────────────────────────────

type DocumentDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mimeType"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateDocumentRequest struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
}
