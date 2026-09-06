package model

import "time"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	Admin AdminDTO `json:"admin"`
}

type TenantPhone struct {
	Type        string `json:"type"`
	CountryCode string `json:"countryCode"`
	Number      string `json:"number"`
	Label       string `json:"label,omitempty"`
}

type TenantStats struct {
	UsersCount   int   `json:"usersCount"`
	ClinicsCount int   `json:"clinicsCount"`
	StorageBytes int64 `json:"storageBytes"`
}

type TenantDTO struct {
	ID                string        `json:"id"`
	Slug              string        `json:"slug"`
	Name              string        `json:"name"`
	DBName            string        `json:"dbName"`
	Phone             string        `json:"phone"`
	PhoneCountry      string        `json:"phoneCountry"`
	Phones            []TenantPhone `json:"phones"`
	MaxClinics        int           `json:"maxClinics"`
	StorageQuotaGB    int           `json:"storageQuotaGB"`
	Modules           []string      `json:"modules"`
	SubscriptionStart *time.Time    `json:"subscriptionStart,omitempty"`
	SubscriptionEnd   *time.Time    `json:"subscriptionEnd,omitempty"`
	IsActive          bool          `json:"isActive"`
	CreatedAt         time.Time     `json:"createdAt"`
	WelcomeSentAt     *time.Time    `json:"welcomeSentAt,omitempty"`
	Stats             TenantStats   `json:"stats"`
}

type CreateTenantRequest struct {
	Slug              string        `json:"slug"`
	Name              string        `json:"name"`
	Phone             string        `json:"phone"`
	PhoneCountry      string        `json:"phoneCountry"`
	Phones            []TenantPhone `json:"phones"`
	AdminEmail        string        `json:"adminEmail"`
	AdminPassword     string        `json:"adminPassword"`
	AdminFullName     string        `json:"adminFullName"`
	AdminPhone        string        `json:"adminPhone"`
	AdminPhoneCountry string        `json:"adminPhoneCountry"`
	MaxClinics        int           `json:"maxClinics"`
	StorageQuotaGB    int           `json:"storageQuotaGB"`
	Modules           []string      `json:"modules"`
	SubscriptionStart string        `json:"subscriptionStart"`
	SubscriptionEnd   string        `json:"subscriptionEnd"`
}

type UpdateTenantRequest struct {
	Name              *string        `json:"name,omitempty"`
	Phone             *string        `json:"phone,omitempty"`
	PhoneCountry      *string        `json:"phoneCountry,omitempty"`
	Phones            *[]TenantPhone `json:"phones,omitempty"`
	MaxClinics        *int           `json:"maxClinics,omitempty"`
	StorageQuotaGB    *int           `json:"storageQuotaGB,omitempty"`
	Modules           *[]string      `json:"modules,omitempty"`
	SubscriptionStart *string        `json:"subscriptionStart,omitempty"`
	SubscriptionEnd   *string        `json:"subscriptionEnd,omitempty"`
	IsActive          *bool          `json:"isActive,omitempty"`
}

type SuperAdminRow struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
}
