package router

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/basicauth"
	swaggerfiles "github.com/swaggo/files/v2"

	"github.com/azhkadaes/vms/backend/config"
	vmsdocs "github.com/azhkadaes/vms/backend/docs"
)

func SwaggerRouter(app *fiber.App, cfg config.Config) {
	if !cfg.SwaggerEnabled {
		return
	}
	username, password := sha256.Sum256([]byte(cfg.SwaggerUsername)), sha256.Sum256([]byte(cfg.SwaggerPassword))
	group := app.Group("/swagger", basicauth.New(basicauth.Config{
		Authorizer: func(user, pass string, _ fiber.Ctx) bool {
			suppliedUser, suppliedPass := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(pass))
			return subtle.ConstantTimeCompare(username[:], suppliedUser[:])&subtle.ConstantTimeCompare(password[:], suppliedPass[:]) == 1
		},
	}))
	group.Get("/openapi.json", func(c fiber.Ctx) error {
		c.Type("json")
		return c.Send(vmsdocs.OpenAPI)
	})
	group.Get("/index.html", func(c fiber.Ctx) error {
		c.Type("html")
		return c.SendString(swaggerHTML)
	})
	group.Get("/*", adaptor.HTTPHandler(http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerfiles.FS)))))
}

const swaggerHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>VMS API</title>
<link rel="stylesheet" href="./swagger-ui.css"></head><body><div id="swagger-ui"></div>
<script src="./swagger-ui-bundle.js"></script><script>
SwaggerUIBundle({url:"./openapi.json",dom_id:"#swagger-ui",validatorUrl:null,persistAuthorization:false});
</script></body></html>`
