package app

import (
	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/middleware"
)

func registerRoutes(router *gin.Engine, h HandlerContainer, s ServiceContainer) {
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/google", h.Auth.GoogleLogin)

			authProtected := auth.Group("")
			authProtected.Use(middleware.RequireAuth(s.Auth))
			authProtected.GET("/me", h.Auth.Me)
			authProtected.POST("/logout", h.Auth.Logout)
		}

		protected := v1.Group("")
		protected.Use(middleware.RequireAuth(s.Auth))
		protected.GET("/sessions/me/active", h.Session.GetActiveSessions)
		protected.GET("/sessions/me", h.Session.ListMySessions)
		protected.GET("/bills/me", h.DailyBill.GetDailyBill)
		protected.GET("/bills/me/pending", h.DailyBill.GetPendingBills)
		protected.POST("/sessions/checkin", h.Session.CheckIn)
		protected.POST("/sessions/checkout", h.Session.CheckOut)
		protected.POST("/boardings/board", h.Boarding.BoardRoute)
		protected.POST("/boardings/auto-board", h.Boarding.AutoDetectAndBoard)
		protected.POST("/boardings/alight", h.Boarding.AlightRoute)
		protected.POST("/boardings/continuous-location", h.Boarding.UpdateContinuousLocation)
		protected.POST("/boardings/tracking-heartbeat", h.Boarding.TrackingHeartbeat)
		protected.GET("/boardings/sessions/:session_id", h.Boarding.GetSessionBoardings)
		protected.GET("/boardings/sessions/:session_id/active", h.Boarding.GetActiveBoarding)
		protected.GET("/bills/users/:user_id", h.DailyBill.GetDailyBill)
		protected.GET("/bills/users/:user_id/pending", h.DailyBill.GetPendingBills)
		protected.POST("/bills/:bill_id/pay", h.DailyBill.PayBill)
		protected.POST("/bills/generate", h.DailyBill.GenerateDailyBills)
		protected.GET("/sessions/users/:user_id/active", h.Session.GetActiveSessions)

		users := v1.Group("/users")
		{
			users.POST("", h.User.CreateUser)
			users.GET("/:id", h.User.GetUser)
			users.GET("/phone/:phone", h.User.GetUserByPhone)
			users.DELETE("/phone/:phone", h.User.DeleteUserByPhone)
		}

		v1.POST("/journeys/plan", h.Journey.PlanJourney)
		v1.GET("/places/search", h.Place.Search)
		v1.GET("/places/resolve", h.Place.Resolve)

		stops := v1.Group("/stops")
		{
			stops.GET("", h.Stop.ListStops)
			stops.GET("/search", h.Stop.SearchStops)
			stops.GET("/nearby", h.Stop.FindNearby)
			stops.GET("/:id", h.Stop.GetStop)
			stops.GET("/:id/departures", h.Stop.GetDepartures)
		}

		routes := v1.Group("/routes")
		{
			routes.GET("", h.Route.ListRoutes)
			routes.GET("/search", h.Route.SearchRoutes)
			routes.GET("/:id", h.Route.GetRoute)
			routes.GET("/:id/detail", h.Route.GetRouteDetail)
			routes.GET("/:id/stops", h.Route.GetRouteStops)
			routes.GET("/:id/trips", h.Route.GetRouteTrips)
		}

		realtime := v1.Group("/realtime")
		{
			realtime.GET("/stops/:id", h.Realtime.GetStopRealtime)
			realtime.GET("/trips/:id", h.Realtime.GetTripRealtime)
		}

		vehicles := v1.Group("/vehicles")
		{
			vehicles.POST("/mock", h.VehicleLocation.AddMockVehicle)
			vehicles.GET("/:vehicle_id", h.VehicleLocation.GetVehicleLocation)
		}

		fares := v1.Group("/fares")
		{
			fares.GET("/calculate", h.Fare.CalculateFare)
			fares.GET("/routes/:id", h.Fare.GetRouteFare)
		}

		sessions := v1.Group("/sessions")
		{
			sessions.POST("/validate-qr", h.Session.ValidateQR)
		}
	}

	v3 := router.Group("/v3")
	{
		v3.GET("/locations", h.V3.Locations)
		v3.GET("/journey", h.V3.Journey)
		v3.GET("/stationboard", h.V3.Stationboard)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
