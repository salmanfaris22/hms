package model

import "time"

type StaffDTO struct {
	ID                 string     `json:"id"`
	StaffCode          string     `json:"staffCode"`
	Email              string     `json:"email"`
	FullName           string     `json:"fullName"`
	FirstName          string     `json:"firstName"`
	MiddleName         string     `json:"middleName"`
	LastName           string     `json:"lastName"`
	ProfilePhoto       string     `json:"profilePhoto"`
	DepartmentID       string     `json:"departmentId,omitempty"`
	DepartmentName     string     `json:"departmentName"`
	DesignationID      string     `json:"designationId,omitempty"`
	DesignationName    string     `json:"designationName"`
	SpecializationID   string     `json:"specializationId,omitempty"`
	SpecializationName string     `json:"specializationName"`
	MobileCountry      string     `json:"mobileCountry"`
	MobileNumber       string     `json:"mobileNumber"`
	AdditionalMobile   string     `json:"additionalMobile"`
	LandlineNumber     string     `json:"landlineNumber"`
	ViewInEMR          bool       `json:"viewInEmr"`
	JoinDate           time.Time  `json:"joinDate"`
	Status             string     `json:"status"`
	IsActive           bool       `json:"isActive"`
	WorkLocations      []string   `json:"workLocations"`
	FatherName         string     `json:"fatherName"`
	MotherName         string     `json:"motherName"`
	Gender             string     `json:"gender"`
	MaritalStatus      string     `json:"maritalStatus"`
	DateOfBirth        *time.Time `json:"dateOfBirth,omitempty"`
	BloodGroup         string     `json:"bloodGroup"`
	ProfessionalID     string     `json:"professionalId"`
	Address            string     `json:"address"`
	Locality           string     `json:"locality"`
	Pincode            string     `json:"pincode"`
	State              string     `json:"state"`
	Country            string     `json:"country"`
	DateOfLeaving      *time.Time `json:"dateOfLeaving,omitempty"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	HasCredentials     bool       `json:"hasCredentials"`
	IsSuperAdmin       bool       `json:"isSuperAdmin"`
	// Comma-joined names of the roles assigned in this clinic; "" when none.
	RoleNames string `json:"roleNames"`
}

type SetCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Required when a user changes their own password; ignored when an
	// administrator sets someone else's, since they cannot know it.
	CurrentPassword string   `json:"currentPassword"`
	RoleID          string   `json:"roleId"`
	ClinicIDs       []string `json:"clinicIds"`
}

type ListFilter struct {
	ClinicID     string
	Page         int
	Limit        int
	Search       string
	DepartmentID string
	Status       string
	AllUsers     bool
}

type CreateStaffRequest struct {
	StaffCode        string   `json:"staffCode"`
	AutoGenerateCode bool     `json:"autoGenerateCode"`
	Email            string   `json:"email"`
	Password         string   `json:"password"`
	FirstName        string   `json:"firstName"`
	MiddleName       string   `json:"middleName"`
	LastName         string   `json:"lastName"`
	ProfilePhoto     string   `json:"profilePhoto"`
	DepartmentID     string   `json:"departmentId"`
	DesignationID    string   `json:"designationId"`
	SpecializationID string   `json:"specializationId"`
	MobileCountry    string   `json:"mobileCountry"`
	MobileNumber     string   `json:"mobileNumber"`
	AdditionalMobile string   `json:"additionalMobile"`
	LandlineNumber   string   `json:"landlineNumber"`
	ViewInEMR        bool     `json:"viewInEmr"`
	WorkLocations    []string `json:"workLocations"`
	FatherName       string   `json:"fatherName"`
	MotherName       string   `json:"motherName"`
	Gender           string   `json:"gender"`
	MaritalStatus    string   `json:"maritalStatus"`
	DateOfBirth      string   `json:"dateOfBirth"`
	BloodGroup       string   `json:"bloodGroup"`
	ProfessionalID   string   `json:"professionalId"`
	Address          string   `json:"address"`
	Locality         string   `json:"locality"`
	Pincode          string   `json:"pincode"`
	State            string   `json:"state"`
	Country          string   `json:"country"`
	DateOfLeaving    string   `json:"dateOfLeaving"`
}

type UpdateStaffRequest struct {
	FirstName        *string   `json:"firstName,omitempty"`
	MiddleName       *string   `json:"middleName,omitempty"`
	LastName         *string   `json:"lastName,omitempty"`
	ProfilePhoto     *string   `json:"profilePhoto,omitempty"`
	DepartmentID     *string   `json:"departmentId,omitempty"`
	DesignationID    *string   `json:"designationId,omitempty"`
	SpecializationID *string   `json:"specializationId,omitempty"`
	MobileCountry    *string   `json:"mobileCountry,omitempty"`
	MobileNumber     *string   `json:"mobileNumber,omitempty"`
	AdditionalMobile *string   `json:"additionalMobile,omitempty"`
	LandlineNumber   *string   `json:"landlineNumber,omitempty"`
	ViewInEMR        *bool     `json:"viewInEmr,omitempty"`
	Status           *string   `json:"status,omitempty"`
	WorkLocations    *[]string `json:"workLocations,omitempty"`
	FatherName       *string   `json:"fatherName,omitempty"`
	MotherName       *string   `json:"motherName,omitempty"`
	Gender           *string   `json:"gender,omitempty"`
	MaritalStatus    *string   `json:"maritalStatus,omitempty"`
	DateOfBirth      *string   `json:"dateOfBirth,omitempty"`
	BloodGroup       *string   `json:"bloodGroup,omitempty"`
	ProfessionalID   *string   `json:"professionalId,omitempty"`
	Address          *string   `json:"address,omitempty"`
	Locality         *string   `json:"locality,omitempty"`
	Pincode          *string   `json:"pincode,omitempty"`
	State            *string   `json:"state,omitempty"`
	Country          *string   `json:"country,omitempty"`
	DateOfLeaving    *string   `json:"dateOfLeaving,omitempty"`
}

type ScheduleSlot struct {
	ID        string `json:"id"`
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	IsActive  bool   `json:"isActive"`
}

type UpdateScheduleRequest struct {
	Slots []ScheduleSlot `json:"slots"`
}

type StatsDTO struct {
	TotalStaff  int `json:"totalStaff"`
	ActiveToday int `json:"activeToday"`
	OnLeave     int `json:"onLeave"`
	Departments int `json:"departments"`
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
