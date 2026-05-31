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

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(middleware.CORSConfig()))

	api := router.Group("/api/v1")
	health.RegisterRoutes(api, db)

	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	jwtAuth := middleware.JWTAuth(cfg.JWTSecret)
	auth.RegisterRoutes(api, authService, jwtAuth)

	admin := api.Group("/admin")
	admin.Use(jwtAuth, middleware.RequireRole("admin"))

	categoryRepo := category.NewRepository(db)
	category.RegisterRoutes(api, admin, categoryRepo)

	productRepo := product.NewRepository(db)
	product.RegisterRoutes(api, admin, productRepo)

	cartRepo := cart.NewRepository(db)
	cartService := cart.NewService(cartRepo)
	cart.RegisterRoutes(api, jwtAuth, cartService)

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
