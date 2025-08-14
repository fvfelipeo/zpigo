// @title           ZPigo WhatsApp API
// @version         1.0
// @description     API para gerenciamento de sessões WhatsApp usando a biblioteca whatsmeow
// @termsOfService  http://swagger.io/terms/

// @contact.name   Suporte da API
// @contact.url    http://www.swagger.io/support
// @contact.email  support@zpigo.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Digite "Bearer " seguido do seu token JWT

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
package main

import (
	"log"
	"os"

	"zpigo/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("Erro ao criar aplicação: %v", err)
	}

	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("Erro ao fechar aplicação: %v", err)
		}
	}()

	if err := application.Run(); err != nil {
		log.Printf("Erro ao executar aplicação: %v", err)
		os.Exit(1)
	}
}
