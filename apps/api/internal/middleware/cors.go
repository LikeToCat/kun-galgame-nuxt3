package middleware

import "github.com/gofiber/fiber/v2/middleware/cors"

func CORS(allowOrigins string) cors.Config {
	return cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
		MaxAge:           86400,
	}
}
