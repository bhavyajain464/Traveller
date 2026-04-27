package app

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/handlers"
	"indian-transit-backend/internal/middleware"
	"indian-transit-backend/internal/repository"
	"indian-transit-backend/internal/services"
)

type App struct {
	Config   *config.Config
	DB       *database.DB
	Redis    *redis.Client
	Services ServiceContainer
	Handlers HandlerContainer
}

type ServiceContainer struct {
	User            *services.UserService
	Auth            *services.AuthService
	Stop            *services.StopService
	PlaceSearch     *services.PlaceSearchService
	V3Journey       *services.V3JourneyService
	Route           *services.RouteService
	Fare            *services.FareService
	JourneyPlanner  services.JourneyPlanner
	JourneySession  *services.JourneySessionService
	VehicleLocation *services.VehicleLocationService
	RouteBoarding   *services.RouteBoardingService
	AutoAlight      *services.AutoAlightService
	DailyBill       *services.DailyBillService
	Realtime        *services.RealtimeService
}

type HandlerContainer struct {
	User            *handlers.UserHandler
	Auth            *handlers.AuthHandler
	Journey         *handlers.JourneyHandler
	Stop            *handlers.StopHandler
	Place           *handlers.PlaceHandler
	V3              *handlers.V3Handler
	Route           *handlers.RouteHandler
	Realtime        *handlers.RealtimeHandler
	Fare            *handlers.FareHandler
	Session         *handlers.JourneySessionHandler
	Boarding        *handlers.RouteBoardingHandler
	VehicleLocation *handlers.VehicleLocationHandler
	DailyBill       *handlers.DailyBillHandler
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	var redisClient *redis.Client
	redisClient, err = config.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Printf("Warning: failed to connect to Redis: %v (continuing without cache)", err)
		redisClient = nil
	}

	if err := services.EnsureAuthSchema(db); err != nil {
		_ = db.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, fmt.Errorf("ensure auth schema: %w", err)
	}

	if err := services.EnsurePhaseOneJourneySchema(db); err != nil {
		_ = db.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, fmt.Errorf("ensure phase 1 journey schema: %w", err)
	}

	serviceContainer := buildServices(db, redisClient, cfg)
	handlerContainer := buildHandlers(serviceContainer)

	return &App{
		Config:   cfg,
		DB:       db,
		Redis:    redisClient,
		Services: serviceContainer,
		Handlers: handlerContainer,
	}, nil
}

func buildServices(db *database.DB, redisClient *redis.Client, cfg *config.Config) ServiceContainer {
	journeySessionRepo := repository.NewJourneySessionRepository(db)
	routeBoardingRepo := repository.NewRouteBoardingRepository(db)
	journeySegmentRepo := repository.NewJourneySegmentRepository(db)
	journeyEventRepo := repository.NewJourneyEventRepository(db)
	fareTransactionRepo := repository.NewFareTransactionRepository(db)
	dailyBillRepo := repository.NewDailyBillRepository(db)
	fareRepo := repository.NewFareRepository(db)
	userService := services.NewUserService(db)
	authService := services.NewAuthService(db, userService, cfg.Auth)
	stopService := services.NewStopService(db)
	placeSearch := buildPlaceSearchService(stopService, cfg)
	routeService := services.NewRouteService(db)
	fareService := services.NewFareService(fareRepo)
	journeyPlanner, err := buildJourneyPlanner(db, stopService, routeService, fareService, cfg)
	if err != nil {
		log.Printf("Warning: failed to initialize configured planner adapter %q: %v", cfg.Planner.Adapter, err)
		sqlAdapter := services.NewSQLJourneyPlannerAdapter(db, stopService, routeService, fareService)
		journeyPlanner = services.NewJourneyPlannerService(sqlAdapter)
	}
	log.Printf("Journey planner engine: %s", journeyPlanner.Engine())
	realtime := services.NewRealtimeService(db, redisClient)
	v3Journey := services.NewV3JourneyService(journeyPlanner, placeSearch, routeService, realtime)
	journeySession := services.NewJourneySessionService(db, journeySessionRepo, dailyBillRepo, journeyEventRepo, stopService, fareService, journeyPlanner)
	vehicleLocation := services.NewVehicleLocationService(db, routeService, stopService)
	routeBoarding := services.NewRouteBoardingService(routeBoardingRepo, journeySegmentRepo, journeyEventRepo, fareTransactionRepo, stopService, fareService, journeySession, vehicleLocation)
	autoAlight := services.NewAutoAlightService(routeBoarding, vehicleLocation, stopService)
	dailyBill := services.NewDailyBillService(dailyBillRepo, journeySessionRepo, fareTransactionRepo)


	return ServiceContainer{
		User:            userService,
		Auth:            authService,
		Stop:            stopService,
		PlaceSearch:     placeSearch,
		V3Journey:       v3Journey,
		Route:           routeService,
		Fare:            fareService,
		JourneyPlanner:  journeyPlanner,
		JourneySession:  journeySession,
		VehicleLocation: vehicleLocation,
		RouteBoarding:   routeBoarding,
		AutoAlight:      autoAlight,
		DailyBill:       dailyBill,
		Realtime:        realtime,
	}
}

func buildJourneyPlanner(db *database.DB, stopService *services.StopService, routeService *services.RouteService, fareService *services.FareService, cfg *config.Config) (services.JourneyPlanner, error) {
	sqlAdapter := services.NewSQLJourneyPlannerAdapter(db, stopService, routeService, fareService)

	switch cfg.Planner.Adapter {
	case "", "in_memory":
		inMemoryAdapter, err := services.NewInMemoryJourneyPlannerAdapter(db, sqlAdapter, fareService)
		if err != nil {
			return nil, err
		}
		return services.NewJourneyPlannerService(inMemoryAdapter), nil
	case "sql":
		return services.NewJourneyPlannerService(sqlAdapter), nil
	default:
		return nil, fmt.Errorf("unsupported planner adapter %q", cfg.Planner.Adapter)
	}
}

func buildPlaceSearchService(stopService *services.StopService, cfg *config.Config) *services.PlaceSearchService {
	fallback := services.NewStopPlaceSearchProvider(stopService)

	switch cfg.PlaceSearch.Provider {
	case "", services.PlaceSearchProviderStopLocal:
		return services.NewPlaceSearchService(fallback)
	case services.PlaceSearchProviderGooglePlaces:
		if cfg.PlaceSearch.GoogleAPIKey == "" {
			log.Printf("Warning: GOOGLE_MAPS_API_KEY is empty; falling back to local stop search provider")
			return services.NewPlaceSearchService(fallback)
		}
		return services.NewPlaceSearchService(services.NewGooglePlacesProvider(cfg.PlaceSearch.GoogleAPIKey, cfg.PlaceSearch.GoogleRegionCode))
	default:
		log.Printf("Warning: unsupported place search provider %q; falling back to local stop search provider", cfg.PlaceSearch.Provider)
		return services.NewPlaceSearchService(fallback)
	}
}

func buildHandlers(s ServiceContainer) HandlerContainer {
	return HandlerContainer{
		User:            handlers.NewUserHandler(s.User),
		Auth:            handlers.NewAuthHandler(s.Auth),
		Journey:         handlers.NewJourneyHandler(s.JourneyPlanner),
		Stop:            handlers.NewStopHandler(s.Stop),
		Place:           handlers.NewPlaceHandler(s.PlaceSearch),
		V3:              handlers.NewV3Handler(s.V3Journey),
		Route:           handlers.NewRouteHandler(s.Route),
		Realtime:        handlers.NewRealtimeHandler(s.Realtime),
		Fare:            handlers.NewFareHandler(s.Fare),
		Session:         handlers.NewJourneySessionHandler(s.JourneySession, s.RouteBoarding),
		Boarding:        handlers.NewRouteBoardingHandler(s.RouteBoarding, s.JourneySession, s.AutoAlight, s.VehicleLocation),
		VehicleLocation: handlers.NewVehicleLocationHandler(s.VehicleLocation),
		DailyBill:       handlers.NewDailyBillHandler(s.DailyBill),
	}
}

func (a *App) Router() *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORS())

	registerRoutes(router, a.Handlers, a.Services)
	return router
}

func (a *App) Run() error {
	addr := fmt.Sprintf("%s:%s", a.Config.Server.Host, a.Config.Server.Port)
	log.Printf("Server starting on %s", addr)
	return a.Router().Run(addr)
}

func (a *App) Close() error {
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			return err
		}
	}
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
