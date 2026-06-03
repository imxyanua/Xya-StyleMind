package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/cart"
	"stylemind/internal/category"
	"stylemind/internal/config"
	"stylemind/internal/database"
	"stylemind/internal/health"
	"stylemind/internal/middleware"
	"stylemind/internal/order"
	"stylemind/internal/product"
	"stylemind/internal/review"
	"stylemind/internal/wishlist"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	rateLimitStore, redisConfigured, err := middleware.NewRateLimitStoreFromConfig(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("rate limit store configuration failed: %v", err)
	}
	defer rateLimitStore.Close()
	if redisConfigured {
		log.Printf("rate limiter using redis at %s", cfg.Redis.Addr)
	} else {
		log.Printf("rate limiter using in-memory fallback; set REDIS_ADDR for multi-instance deployments")
	}
	authRateLimit := middleware.NewRateLimiter(rateLimitStore, 10, time.Minute, middleware.WithFailClosed(redisConfigured))

	tokenRevocationStore, tokenRevocationRedisConfigured, err := auth.NewTokenRevocationStoreFromConfig(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("token revocation store configuration failed: %v", err)
	}
	defer tokenRevocationStore.Close()
	if tokenRevocationRedisConfigured {
		log.Printf("token revocation using redis at %s", cfg.Redis.Addr)
	} else {
		log.Printf("token revocation using in-memory fallback; set REDIS_ADDR for multi-instance deployments")
	}
	var redisHealth health.Pinger
	if redisConfigured {
		redisChecker, err := health.NewRedisChecker(cfg.Redis)
		if err != nil {
			log.Fatalf("redis health checker configuration failed: %v", err)
		}
		defer redisChecker.Close()
		redisHealth = redisChecker
	}

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to configure trusted proxies: %v", err)
	}
	httpMetrics := middleware.NewHTTPMetrics()
	router.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		httpMetrics.Middleware(),
		middleware.RequestTimeout(time.Duration(cfg.RequestTimeoutSeconds)*time.Second),
		middleware.RequestLogger(),
		gin.Recovery(),
		middleware.RequestBodyLimit(cfg.MaxRequestBodyBytes),
	)
	router.Use(cors.New(middleware.CORSConfig(cfg.CORSAllowedOrigins)))
	httpMetrics.RegisterRoutes(router)

	api := router.Group("/api/v1")
	health.RegisterRoutes(router, api, health.NewPostgresChecker(db), redisHealth, redisConfigured)

	tokenConfig := auth.TokenConfig{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
	}
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, tokenConfig, auth.WithTokenRevocationStore(tokenRevocationStore))
	jwtAuth := middleware.JWTAuth(tokenConfig, middleware.WithTokenRevocationStore(tokenRevocationStore))
	auth.RegisterRoutes(
		api,
		authService,
		jwtAuth,
		authRateLimit.Middleware("auth:login", middleware.IPKeyExtractor, middleware.EmailKeyExtractor),
		authRateLimit.Middleware("auth:register", middleware.IPKeyExtractor, middleware.EmailKeyExtractor),
	)

	admin := api.Group("/admin")
	admin.Use(jwtAuth, middleware.RequireRole("admin"))

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo)
	category.RegisterRoutes(api, admin, categoryService)

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo)
	product.RegisterRoutes(api, admin, productService)

	reviewRepo := review.NewRepository(db)
	reviewService := review.NewService(reviewRepo)
	review.RegisterRoutes(api, jwtAuth, reviewService)

	cartRepo := cart.NewRepository(db)
	cartService := cart.NewService(cartRepo)
	cart.RegisterRoutes(api, jwtAuth, cartService)

	wishlistRepo := wishlist.NewRepository(db)
	wishlistService := wishlist.NewService(wishlistRepo)
	wishlist.RegisterRoutes(api, jwtAuth, wishlistService)

	orderRepo := order.NewRepository(db)
	orderService := order.NewService(orderRepo)
	order.RegisterRoutes(api, admin, jwtAuth, orderService)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
