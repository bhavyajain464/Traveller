package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/handlers"
	"indian-transit-backend/internal/middleware"
	"indian-transit-backend/internal/services"
)

func main() {
	cfg := config.Load()

	// Initialize database
	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis (optional - service will work without it)
	var redisClient *redis.Client
	redisClient, err = config.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v (continuing without cache)", err)
		redisClient = nil
	}

	// Initialize services
	userService := services.NewUserService(db)
	stopService := services.NewStopService(db)
	routeService := services.NewRouteService(db)
	fareService := services.NewFareService(db)
	routePlanner := services.NewRoutePlanner(db, stopService, routeService, fareService)
	journeySessionService := services.NewJourneySessionService(db, stopService, fareService, routePlanner)
	vehicleLocationService := services.NewVehicleLocationService(db, routeService, stopService)
	routeBoardingService := services.NewRouteBoardingService(db, stopService, fareService, journeySessionService, vehicleLocationService)
	autoAlightService := services.NewAutoAlightService(routeBoardingService, vehicleLocationService, stopService)
	dailyBillService := services.NewDailyBillService(db)

	var realtimeService *services.RealtimeService
	if redisClient != nil {
		realtimeService = services.NewRealtimeService(db, redisClient)
	} else {
		// Create service with nil Redis - will use scheduled times only
		realtimeService = services.NewRealtimeService(db, nil)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	journeyHandler := handlers.NewJourneyHandler(routePlanner)
	stopHandler := handlers.NewStopHandler(stopService)
	routeHandler := handlers.NewRouteHandler(routeService)
	realtimeHandler := handlers.NewRealtimeHandler(realtimeService)
	fareHandler := handlers.NewFareHandler(fareService)
	sessionHandler := handlers.NewJourneySessionHandler(journeySessionService, routeBoardingService)
	boardingHandler := handlers.NewRouteBoardingHandler(routeBoardingService, journeySessionService, autoAlightService, vehicleLocationService)
	vehicleLocationHandler := handlers.NewVehicleLocationHandler(vehicleLocationService)
	billHandler := handlers.NewDailyBillHandler(dailyBillService)

	// Setup router
	router := gin.Default()
	router.Use(middleware.CORS())

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Users
		users := v1.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.GET("/phone/:phone", userHandler.GetUserByPhone)
			users.DELETE("/phone/:phone", userHandler.DeleteUserByPhone)
		}

		// Journey planning
		v1.POST("/journeys/plan", journeyHandler.PlanJourney)

		// Stops
		stops := v1.Group("/stops")
		{
			stops.GET("", stopHandler.ListStops)
			stops.GET("/search", stopHandler.SearchStops)
			stops.GET("/nearby", stopHandler.FindNearby)
			stops.GET("/:id", stopHandler.GetStop)
			stops.GET("/:id/departures", stopHandler.GetDepartures)
		}

		// Routes
		routes := v1.Group("/routes")
		{
			routes.GET("", routeHandler.ListRoutes)
			routes.GET("/search", routeHandler.SearchRoutes)
			routes.GET("/:id", routeHandler.GetRoute)
			routes.GET("/:id/detail", routeHandler.GetRouteDetail)
			routes.GET("/:id/stops", routeHandler.GetRouteStops)
			routes.GET("/:id/trips", routeHandler.GetRouteTrips)
		}

		// Real-time
		realtime := v1.Group("/realtime")
		{
			realtime.GET("/stops/:id", realtimeHandler.GetStopRealtime)
			realtime.GET("/trips/:id", realtimeHandler.GetTripRealtime)
		}

		// Vehicle Locations (for testing/mocking)
		vehicles := v1.Group("/vehicles")
		{
			vehicles.POST("/mock", vehicleLocationHandler.AddMockVehicle)           // Add mock vehicle for testing
			vehicles.GET("/:vehicle_id", vehicleLocationHandler.GetVehicleLocation) // Get vehicle location
		}

		// Fares
		fares := v1.Group("/fares")
		{
			fares.GET("/calculate", fareHandler.CalculateFare)
			fares.GET("/routes/:id", fareHandler.GetRouteFare)
		}

		// Journey Sessions (Check-in/Check-out)
		sessions := v1.Group("/sessions")
		{
			sessions.POST("/checkin", sessionHandler.CheckIn)
			sessions.POST("/checkout", sessionHandler.CheckOut)
			sessions.POST("/validate-qr", sessionHandler.ValidateQR) // Also records boarding
			sessions.GET("/users/:user_id/active", sessionHandler.GetActiveSessions)
		}

		// Route Boardings (Track actual routes user takes)
		boardings := v1.Group("/boardings")
		{
			boardings.POST("/board", boardingHandler.BoardRoute)                             // Record boarding a route (manual)
			boardings.POST("/auto-board", boardingHandler.AutoDetectAndBoard)                // Auto-detect vehicle and board
			boardings.POST("/alight", boardingHandler.AlightRoute)                           // Record alighting from route
			boardings.POST("/continuous-location", boardingHandler.UpdateContinuousLocation) // Continuous location updates with auto-alighting
			boardings.GET("/sessions/:session_id", boardingHandler.GetSessionBoardings)      // Get all boardings for session
			boardings.GET("/sessions/:session_id/active", boardingHandler.GetActiveBoarding) // Get active boarding
		}

		// Daily Bills
		bills := v1.Group("/bills")
		{
			bills.GET("/users/:user_id", billHandler.GetDailyBill)
			bills.GET("/users/:user_id/pending", billHandler.GetPendingBills)
			bills.POST("/:bill_id/pay", billHandler.PayBill)
			bills.POST("/generate", billHandler.GenerateDailyBills) // Admin endpoint
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
