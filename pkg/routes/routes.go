package routes

import (
	"github.com/gofiber/fiber/v2"

	apptHandler "github.com/salman/hms-backend/internal/modules/appointment/handler"
	catalogueHandler "github.com/salman/hms-backend/internal/modules/catalogue/handler"
	clinicHandler "github.com/salman/hms-backend/internal/modules/clinic/handler"
	dashboardHandler "github.com/salman/hms-backend/internal/modules/dashboard/handler"
	inventoryHandler "github.com/salman/hms-backend/internal/modules/inventory/handler"
	logsHandler "github.com/salman/hms-backend/internal/modules/logs/handler"
	patientHandler "github.com/salman/hms-backend/internal/modules/patient/handler"
	staffHandler "github.com/salman/hms-backend/internal/modules/staff/handler"
	superHandler "github.com/salman/hms-backend/internal/modules/super/handler"
	uploadHandler "github.com/salman/hms-backend/internal/modules/upload/handler"
	userHandler "github.com/salman/hms-backend/internal/modules/user/handler"
)

type AuthHandlers struct {
	Auth         *userHandler.Handler
	Clinics      *clinicHandler.Handler
	Patients     *patientHandler.Handler
	Appointments *apptHandler.Handler
	Catalogue    *catalogueHandler.Handler
	Super        *superHandler.Handler
	Uploads      *uploadHandler.Handler
	Inventory    *inventoryHandler.Handler
}

func Register(app *fiber.App, h AuthHandlers) {
	AuthRoutes(app, h.Auth)
	ClinicsRoutes(app, h.Clinics)
	PatientsRoutes(app, h.Patients)
	AppointmentsRoutes(app, h.Appointments)
	CatalogueRoutes(app, h.Catalogue)
	SuperRoutes(app, h.Super)
	UploadsRoutes(app, h.Uploads)
	InventoryRoutes(app, h.Inventory)
}

func AuthRoutes(r fiber.Router, h *userHandler.Handler) {
	r.Post("/login", h.Login)
	r.Get("/me", h.Me)
}

func ClinicsRoutes(r fiber.Router, h *clinicHandler.Handler) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Post("/join", h.Join)
	r.Get("/:id", h.Get)
	r.Patch("/:id", h.Update)
	r.Delete("/:id", h.Delete)
	r.Post("/:id/archive", h.Archive)
	r.Post("/:id/restore", h.Restore)
	r.Post("/:id/primary", h.MakePrimary)
	r.Get("/:id/numbering", h.GetNumberingSettings)
	r.Put("/:id/numbering", h.PutNumberingSettings)
	r.Get("/:id/print", h.GetPrintSettings)
	r.Put("/:id/print", h.PutPrintSettings)
	r.Get("/:id/communication", h.GetCommunicationSettings)
	r.Put("/:id/communication", h.PutCommunicationSettings)
	r.Get("/:id/taxes", h.ListTaxes)
	r.Post("/:id/taxes", h.CreateTax)
	r.Patch("/:id/taxes/:taxId", h.UpdateTax)
	r.Delete("/:id/taxes/:taxId", h.DeleteTax)
	r.Get("/:id/payment-modes", h.ListPaymentModes)
	r.Post("/:id/payment-modes", h.CreatePaymentMode)
	r.Delete("/:id/payment-modes/:modeId", h.DeletePaymentMode)
	r.Get("/:id/patient-categories", h.ListPatientCategories)
	r.Post("/:id/patient-categories", h.CreatePatientCategory)
	r.Patch("/:id/patient-categories/:catId", h.UpdatePatientCategory)
	r.Delete("/:id/patient-categories/:catId", h.DeletePatientCategory)
	r.Get("/:id/special-statuses", h.ListSpecialStatuses)
	r.Post("/:id/special-statuses", h.CreateSpecialStatus)
	r.Patch("/:id/special-statuses/:statusId", h.UpdateSpecialStatus)
	r.Delete("/:id/special-statuses/:statusId", h.DeleteSpecialStatus)
	r.Get("/:id/integration", h.GetIntegrationSettings)
	r.Put("/:id/integration", h.PutIntegrationSettings)

	r.Post("/:id/org-setup/copy", h.CopyOrgSetup)

	r.Get("/:id/departments", h.ListDepartments)
	r.Post("/:id/departments", h.CreateDepartment)
	r.Patch("/:id/departments/:itemId", h.UpdateDepartment)
	r.Delete("/:id/departments/:itemId", h.DeleteDepartment)

	r.Get("/:id/specializations", h.ListSpecializations)
	r.Post("/:id/specializations", h.CreateSpecialization)
	r.Patch("/:id/specializations/:itemId", h.UpdateSpecialization)
	r.Delete("/:id/specializations/:itemId", h.DeleteSpecialization)

	r.Get("/:id/designations", h.ListDesignations)
	r.Post("/:id/designations", h.CreateDesignation)
	r.Patch("/:id/designations/:itemId", h.UpdateDesignation)
	r.Delete("/:id/designations/:itemId", h.DeleteDesignation)
}

func PatientsRoutes(r fiber.Router, h *patientHandler.Handler) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/field-config", h.GetFieldConfig)
	r.Put("/field-config", h.PutFieldConfig)
	r.Get("/:id", h.Get)
	r.Get("/:id/profile", h.GetProfile)
	r.Post("/:id/seed-demo", h.SeedDemo)
	r.Post("/:id/alerts", h.AddAlert)
	r.Post("/:id/allergies", h.AddAllergy)
	r.Delete("/:id", h.Delete)
	r.Delete("/alerts/:id", h.DeleteAlert)
	r.Delete("/allergies/:id", h.DeleteAllergy)
}

func AppointmentsRoutes(r fiber.Router, h *apptHandler.Handler) {
	r.Post("/", h.Book)
	r.Patch("/:id", h.Update)
	r.Get("/settings", h.GetSettings)
	r.Put("/settings", h.PutSettings)
}

func CatalogueRoutes(r fiber.Router, h *catalogueHandler.Handler) {
	r.Get("/drugs", h.ListDrugs)
	r.Post("/drugs", h.CreateDrug)
	r.Patch("/drugs/:id", h.UpdateDrug)
	r.Delete("/drugs/:id", h.DeleteDrug)

	r.Get("/treatments", h.ListTreatments)
	r.Post("/treatments", h.CreateTreatment)
	r.Patch("/treatments/:id", h.UpdateTreatment)
	r.Delete("/treatments/:id", h.DeleteTreatment)

	r.Get("/radiology/tests", h.ListRadiologyTests)
	r.Post("/radiology/tests", h.CreateRadiologyTest)
	r.Patch("/radiology/tests/:id", h.UpdateRadiologyTest)
	r.Delete("/radiology/tests/:id", h.DeleteRadiologyTest)
	r.Get("/radiology/categories", h.ListRadiologyCategories)
	r.Post("/radiology/categories", h.CreateRadiologyCategory)
	r.Delete("/radiology/categories/:id", h.DeleteRadiologyCategory)

	r.Get("/pathology/tests", h.ListPathologyTests)
	r.Post("/pathology/tests", h.CreatePathologyTest)
	r.Patch("/pathology/tests/:id", h.UpdatePathologyTest)
	r.Put("/pathology/tests/:id/parameters", h.SyncPathologyTestParameters)
	r.Delete("/pathology/tests/:id", h.DeletePathologyTest)
	r.Get("/pathology/categories", h.ListPathologyCategories)
	r.Post("/pathology/categories", h.CreatePathologyCategory)
	r.Delete("/pathology/categories/:id", h.DeletePathologyCategory)
	r.Get("/pathology/parameters", h.ListPathologyParameters)
	r.Post("/pathology/parameters", h.CreatePathologyParameter)
	r.Patch("/pathology/parameters/:id", h.UpdatePathologyParameter)
	r.Delete("/pathology/parameters/:id", h.DeletePathologyParameter)

	r.Get("/pathology/groups", h.ListPathologyGroups)
	r.Post("/pathology/groups", h.CreatePathologyGroup)
	r.Patch("/pathology/groups/:id", h.UpdatePathologyGroup)
	r.Delete("/pathology/groups/:id", h.DeletePathologyGroup)
	r.Delete("/pathology/groups/:id/parameters/:paramId", h.RemovePathologyGroupParam)

	r.Post("/copy", h.CopyCatalogue)

	r.Get("/categories", h.ListCategories)
	r.Post("/categories", h.CreateCategory)
	r.Delete("/categories/:id", h.DeleteCategory)
	r.Get("/manufacturers", h.ListManufacturers)
	r.Post("/manufacturers", h.CreateManufacturer)
	r.Delete("/manufacturers/:id", h.DeleteManufacturer)
	r.Get("/units", h.ListUnits)
	r.Post("/units", h.CreateUnit)
	r.Delete("/units/:id", h.DeleteUnit)
}

func SuperRoutes(r fiber.Router, h *superHandler.Handler) {
	r.Post("/auth/login", h.Login)

	r.Get("/tenants", h.ListTenants)
	r.Post("/tenants", h.CreateTenant)
	r.Patch("/tenants/:id", h.UpdateTenant)
	r.Delete("/tenants/:id", h.DeleteTenant)
	r.Post("/tenants/:id/subscription-reminder", h.SendSubscriptionReminder)
}

func UploadsRoutes(r fiber.Router, h *uploadHandler.Handler) {
	r.Get("/sign", h.Sign)
}

func LogsRoutes(r fiber.Router, h *logsHandler.Handler) {
	r.Get("/", h.List)
	r.Get("/export", h.Export)
}

func InventoryRoutes(r fiber.Router, h *inventoryHandler.Handler) {
	// drugs — static paths before :id
	r.Get("/inventory/drugs/summary", h.DrugSummary)
	r.Get("/inventory/drugs/pos", h.SearchDrugsForPOS)
	r.Get("/inventory/drugs", h.ListDrugs)
	r.Post("/inventory/drugs", h.CreateDrug)
	r.Delete("/inventory/drugs/:id", h.DeleteDrug)
	// categories & manufacturers
	r.Get("/inventory/categories", h.ListCategories)
	r.Post("/inventory/categories", h.CreateCategory)
	r.Delete("/inventory/categories/:id", h.DeleteCategory)
	r.Get("/inventory/manufacturers", h.ListManufacturers)
	r.Post("/inventory/manufacturers", h.CreateManufacturer)
	r.Delete("/inventory/manufacturers/:id", h.DeleteManufacturer)
	// stock
	r.Get("/inventory/stock", h.ListStock)
	r.Post("/inventory/stock", h.AddStock)
	// sales — static paths before :id
	r.Get("/inventory/sales/summary", h.SalesSummary)
	r.Get("/inventory/sales", h.ListSales)
	r.Post("/inventory/sales", h.CreateSale)
	r.Delete("/inventory/sales/:id", h.DeleteSale)
	// dispenses — static paths before :id
	r.Get("/inventory/dispenses/summary", h.DispenseSummary)
	r.Get("/inventory/dispenses", h.ListDispenses)
	r.Post("/inventory/dispenses", h.CreateDispense)
	r.Patch("/inventory/dispenses/:id", h.UpdateDispenseStatus)
	// assets — static paths before :id
	r.Get("/inventory/assets/summary", h.AssetSummary)
	r.Get("/inventory/assets", h.ListAssets)
	r.Post("/inventory/assets", h.CreateAsset)
	r.Patch("/inventory/assets/:id", h.UpdateAsset)
	r.Delete("/inventory/assets/:id", h.DeleteAsset)
	// purchases
	r.Get("/inventory/purchases", h.ListPurchases)
	r.Post("/inventory/purchases", h.CreatePurchase)
	r.Delete("/inventory/purchases/:id", h.DeletePurchase)
}

func StaffRoutes(r fiber.Router, h *staffHandler.Handler) {
	r.Get("/stats", h.Stats)
	r.Get("/", h.ListStaff)
	r.Post("/", h.CreateStaff)
	r.Get("/:id", h.GetStaff)
	r.Patch("/:id", h.UpdateStaff)
	r.Post("/:id/deactivate", h.DeactivateStaff)
	r.Get("/:id/documents", h.ListDocuments)
	r.Post("/:id/documents", h.CreateDocument)
	r.Delete("/:id/documents/:docId", h.DeleteDocument)
}

func RolesRoutes(r fiber.Router, h *staffHandler.Handler) {
	r.Get("/", h.ListRoles)
	r.Post("/", h.CreateRole)
	r.Patch("/:id", h.UpdateRole)
	r.Delete("/:id", h.DeleteRole)
	r.Post("/assign", h.AssignRoles)
}

func DashboardRoutes(r fiber.Router, h *dashboardHandler.Handler) {
	r.Get("/overview", h.Overview)
}
