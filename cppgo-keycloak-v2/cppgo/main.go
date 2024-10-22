package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"cpp/auth"
	"cpp/config"
	"cpp/db"
	"cpp/handler"
)

func main() {
	config := config.LoadConfig()

	oauthConfig := auth.SetupOAuth2Config(config)

	publicKey, err := auth.GetPublicKey(config.KeycloakURL, config.KeycloakRealm)
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}

	database, err := db.SetupDatabase(config.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	h := handler.NewHandler(database, oauthConfig, config)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/lib", "./lib")

	r.GET("/login", h.LoginHandler)
	r.GET("/callback", h.CallbackHandler)
	r.GET("/logout", h.LogoutHandler)

	r.POST("/callNotifications/v1/networks/:network_id/notifications/services/:service_id/callDirections", h.CallNotificationHandler)
	r.GET("/list", auth.JwtMiddleware(publicKey), h.ListHandler)
	r.POST("/add", auth.JwtMiddleware(publicKey), h.AddHandler)
	r.GET("/", auth.JwtMiddleware(publicKey), h.IndexHandler)
	r.GET("/edit/:id", auth.JwtMiddleware(publicKey), h.EditHandler)
	r.POST("/update/:id", auth.JwtMiddleware(publicKey), h.UpdateHandler)
	r.GET("/delete/:id", auth.JwtMiddleware(publicKey), h.DeleteHandler)
	r.GET("/calls/:phone_number/lasthour", auth.JwtMiddleware(publicKey), h.LastHourCallsHandler)
	r.GET("/calls", auth.JwtMiddleware(publicKey), h.CallLogsHandler)
	r.GET("/monitor", auth.JwtMiddleware(publicKey), h.MonitorHandler)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
