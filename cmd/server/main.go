package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/salman/hms-backend/internal/core/config"
	"github.com/salman/hms-backend/internal/core/middleware"
	"github.com/salman/hms-backend/internal/core/seeder"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	apptHandler "github.com/salman/hms-backend/internal/modules/appointment/handler"
	apptRepo "github.com/salman/hms-backend/internal/modules/appointment/repository"
	apptService "github.com/salman/hms-backend/internal/modules/appointment/service"
	billingHandler "github.com/salman/hms-backend/internal/modules/billing/handler"
	billingRepo "github.com/salman/hms-backend/internal/modules/billing/repository"
	billingService "github.com/salman/hms-backend/internal/modules/billing/service"
	catalogueHandler "github.com/salman/hms-backend/internal/modules/catalogue/handler"
	catalogueRepo "github.com/salman/hms-backend/internal/modules/catalogue/repository"
	catalogueService "github.com/salman/hms-backend/internal/modules/catalogue/service"
	clinicHandler "github.com/salman/hms-backend/internal/modules/clinic/handler"
	clinicRepo "github.com/salman/hms-backend/internal/modules/clinic/repository"
	clinicService "github.com/salman/hms-backend/internal/modules/clinic/service"
	dashboardHandler "github.com/salman/hms-backend/internal/modules/dashboard/handler"
	dashboardRepo "github.com/salman/hms-backend/internal/modules/dashboard/repository"
	dashboardService "github.com/salman/hms-backend/internal/modules/dashboard/service"
	inventoryHandler "github.com/salman/hms-backend/internal/modules/inventory/handler"
	inventoryRepo "github.com/salman/hms-backend/internal/modules/inventory/repository"
	inventoryService "github.com/salman/hms-backend/internal/modules/inventory/service"
	labHandler "github.com/salman/hms-backend/internal/modules/lab/handler"
	labRepo "github.com/salman/hms-backend/internal/modules/lab/repository"
	labService "github.com/salman/hms-backend/internal/modules/lab/service"
	logsHandler "github.com/salman/hms-backend/internal/modules/logs/handler"
	logsRepo "github.com/salman/hms-backend/internal/modules/logs/repository"
	logsService "github.com/salman/hms-backend/internal/modules/logs/service"
	patientHandler "github.com/salman/hms-backend/internal/modules/patient/handler"
	patientRepo "github.com/salman/hms-backend/internal/modules/patient/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
	staffHandler "github.com/salman/hms-backend/internal/modules/staff/handler"
	staffRepo "github.com/salman/hms-backend/internal/modules/staff/repository"
	staffService "github.com/salman/hms-backend/internal/modules/staff/service"
	superHandler "github.com/salman/hms-backend/internal/modules/super/handler"
	superRepo "github.com/salman/hms-backend/internal/modules/super/repository"
	superService "github.com/salman/hms-backend/internal/modules/super/service"
	uploadHandler "github.com/salman/hms-backend/internal/modules/upload/handler"
	uploadService "github.com/salman/hms-backend/internal/modules/upload/service"
	userHandler "github.com/salman/hms-backend/internal/modules/user/handler"
	userRepo "github.com/salman/hms-backend/internal/modules/user/repository"
	userService "github.com/salman/hms-backend/internal/modules/user/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/email"
	"github.com/salman/hms-backend/pkg/routes"
	"github.com/salman/hms-backend/pkg/sms"
	"github.com/salman/hms-backend/pkg/whatsapp"
)

func main() {
	cfg := config.Load()

	dbURL := cfg.DatabaseURL
	if cfg.PgBouncerURL != "" {
		dbURL = cfg.PgBouncerURL
		log.Printf("db: using pgbouncer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resolver, registry, err := postgres.NewTenantResolver(ctx, dbURL)
	if err != nil {
		log.Fatalf("db bootstrap: %v", err)
	}
	defer resolver.Close()

	if err := seeder.EnsureDemoTenant(ctx, resolver); err != nil {
		log.Fatalf("seed demo tenant: %v", err)
	}
	if err := resolver.MigrateAllTenants(ctx); err != nil {
		log.Fatalf("migrate tenants: %v", err)
	}
	if err := seeder.EnsureInventoryData(ctx, resolver); err != nil {
		log.Printf("inventory seed: %v", err)
	}
	if err := seeder.EnsureSuperAdmin(ctx, registry, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		log.Fatalf("seed super admin: %v", err)
	}

	mailer := email.New(cfg.SendGridAPIKey, cfg.EmailFrom, cfg.EmailFromName)

	uRepo := userRepo.New(registry)
	uSvc := userService.New(uRepo, resolver, cfg.JWTSecret)
	authH := userHandler.New(uSvc)
	clRepo := clinicRepo.New(resolver)
	clSvc := clinicService.New(clRepo)
	clinicsH := clinicHandler.New(clSvc)
	pRepo := patientRepo.New(resolver)
	smsSender := sms.New(cfg.SMSAPIKey, cfg.SMSFrom)
	waSender := whatsapp.New(cfg.WhatsAppAPIKey, cfg.WhatsAppFrom)
	pSvc := patientService.New(pRepo, clRepo, mailer, smsSender, waSender)
	patientsH := patientHandler.New(pSvc)
	aRepo := apptRepo.New()
	aSvc := apptService.New(aRepo, pSvc)
	appointmentsH := apptHandler.New(aSvc)
	bRepo := billingRepo.New()
	bSvc := billingService.New(bRepo, pSvc)
	billingH := billingHandler.New(bSvc)

	lbRepo := labRepo.New()
	lbSvc := labService.New(lbRepo, pSvc)
	labH := labHandler.New(lbSvc)
	catRepo := catalogueRepo.New(pSvc)
	catSvc := catalogueService.New(catRepo, pSvc)
	catalogueH := catalogueHandler.New(catSvc)
	sRepo := superRepo.New(registry, resolver)
	sSvc := superService.New(sRepo, cfg.JWTSecret, mailer, cfg.AppURL)
	superH := superHandler.New(sSvc)

	upSvc := uploadService.New(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	uploadsH := uploadHandler.New(upSvc)

	lRepo := logsRepo.New(registry)
	lSvc := logsService.New(lRepo)
	logsH := logsHandler.New(lSvc)
	invRepo := inventoryRepo.New(resolver)
	invSvc := inventoryService.New(invRepo)
	inventoryH := inventoryHandler.New(invSvc)
	stRepo := staffRepo.New(resolver)
	stSvc := staffService.New(stRepo)
	staffH := staffHandler.New(stSvc)
	dashRepo := dashboardRepo.New(resolver)
	dashSvc := dashboardService.New(dashRepo)
	dashH := dashboardHandler.New(dashSvc)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			c.Status(code)
			return c.JSON(fiber.Map{
				"success":    false,
				"statusCode": code,
				"message":    err.Error(),
			})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		// the dev servers by default; a hosted frontend names its own origin
		// through CORS_ORIGINS. Never "*" while AllowCredentials is on.
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Accept,Authorization,Content-Type,X-Tenant",
		AllowCredentials: true,
	}))

	app.Get(constants.APIHealth, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post(constants.APIAuth+"/login", authH.Login)
	app.Get(constants.APIAuth+"/me", middleware.Require(cfg.JWTSecret), authH.Me)

	clinics := app.Group(constants.APIClinics, middleware.Require(cfg.JWTSecret))
	routes.ClinicsRoutes(clinics, clinicsH)

	patients := app.Group(constants.APIPatients, middleware.Require(cfg.JWTSecret))
	routes.PatientsRoutes(patients, patientsH)

	appointments := app.Group(constants.APIAppointments, middleware.Require(cfg.JWTSecret))
	routes.AppointmentsRoutes(appointments, appointmentsH)

	billing := app.Group(constants.APIBilling, middleware.Require(cfg.JWTSecret))
	routes.BillingRoutes(billing, billingH)

	lab := app.Group(constants.APILab, middleware.Require(cfg.JWTSecret))
	routes.LabRoutes(lab, labH)

	catalogue := app.Group(constants.APICatalogue, middleware.Require(cfg.JWTSecret))
	routes.CatalogueRoutes(catalogue, catalogueH)

	superGroup := app.Group(constants.APISuper)
	superGroup.Post("/auth/login", superH.Login)
	superGroup.Use(middleware.Require(cfg.JWTSecret))
	superGroup.Use(middleware.EnforceSuper)
	routes.SuperRoutes(superGroup, superH)

	uploads := app.Group(constants.APIUploads, middleware.Require(cfg.JWTSecret))
	routes.UploadsRoutes(uploads, uploadsH)

	logsGroup := app.Group(constants.APILogs, middleware.Require(cfg.JWTSecret))
	routes.LogsRoutes(logsGroup, logsH)

	invGroup := app.Group("/api", middleware.Require(cfg.JWTSecret))
	routes.InventoryRoutes(invGroup, inventoryH)

	staffGroup := app.Group(constants.APIStaff, middleware.Require(cfg.JWTSecret))
	routes.StaffRoutes(staffGroup, staffH)
	rolesGroup := app.Group(constants.APIRoles, middleware.Require(cfg.JWTSecret))
	routes.RolesRoutes(rolesGroup, staffH)
	dashGroup := app.Group(constants.APIDashboard, middleware.Require(cfg.JWTSecret))
	routes.DashboardRoutes(dashGroup, dashH)

	log.Printf("HMS backend listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
