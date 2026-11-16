package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/services"
	in "perdimidocumento/src/infrastructure/in"
	out "perdimidocumento/src/infrastructure/out/db"
)

func main() {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "data/documentos.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		log.Fatal(err)
	}
	db, err := out.OpenSingleton(path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := &out.SQLiteRepository{DB: db}
	auth := services.AuthService{
		Repository: repo,
		Hasher:     services.BcryptPasswordHasher{},
		Tokens:     services.SHA256TokenProvider{},
	}
	handler := (&in.Server{
		UserRegistrar:    auth,
		SessionManager:   auth,
		Registrar:        services.DocumentProxy{Repository: repo, Factory: models.DocumentFactory{}},
		DocumentSearcher: services.DocumentSearchService{Repository: repo},
		Subscriptions:    services.SubscriptionService{Repository: repo},
		Notifications:    services.NotificationService{Repository: repo},
		SecureCookies:    os.Getenv("COOKIE_SECURE") == "true",
	}).Handler()

	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Escuchando en %s", address)
	log.Fatal(server.ListenAndServe())
}
