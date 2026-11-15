package tests

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
	"perdimidocumento/src/infrastructure/out"
)

func sqliteRepo(t *testing.T) *out.SQLiteRepository {
	t.Helper()
	db, err := out.Open(filepath.Join(t.TempDir(), "integration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &out.SQLiteRepository{DB: db}
}

func TestSQLiteRepositorySessionLifecycle(t *testing.T) {
	repo := sqliteRepo(t)
	ctx := context.Background()

	if err := repo.CreateUser(ctx, "ana", "hash"); err != nil {
		t.Fatal(err)
	}
	user, err := repo.UserByName(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.CreateSession(ctx, user.ID, "hash-token", time.Now().Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	logged, err := repo.UserBySession(ctx, "hash-token")
	if err != nil {
		t.Fatal(err)
	}
	if logged.ID != user.ID || logged.Username != user.Username {
		t.Fatalf("usuario por sesión inesperado: %+v", logged)
	}

	if err := repo.CreateSession(ctx, user.ID, "hash-expirada", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UserBySession(ctx, "hash-expirada"); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("la sesión expirada debe estar ausente: %v", err)
	}

	if err := repo.DeleteSession(ctx, "hash-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UserBySession(ctx, "hash-token"); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("la sesión eliminada debe estar ausente: %v", err)
	}
}

func TestSQLiteRepositorySearchDocumentsReturnsLatestFirst(t *testing.T) {
	repo := sqliteRepo(t)
	ctx := context.Background()

	if err := repo.CreateUser(ctx, "reportero", "hash"); err != nil {
		t.Fatal(err)
	}
	user, err := repo.UserByName(ctx, "reportero")
	if err != nil {
		t.Fatal(err)
	}

	factory := models.DocumentFactory{}
	doc1, err := factory.Create("001", "Cédula", "Ana", "Oficina centro", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := factory.Create("001", "Cédula", "Ana", "Oficina norte", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	created1, err := repo.RegisterDocument(ctx, doc1)
	if err != nil {
		t.Fatal(err)
	}
	created2, err := repo.RegisterDocument(ctx, doc2)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := repo.SearchDocuments(ctx, "001")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("cantidad inesperada: %d", len(docs))
	}
	if docs[0].ID != created2.ID || docs[0].Address != "Oficina norte" {
		t.Fatalf("el documento más reciente debe ir primero: %+v", docs[0])
	}
	if docs[1].ID != created1.ID || docs[1].Address != "Oficina centro" {
		t.Fatalf("el documento anterior debe ir al final: %+v", docs[1])
	}
}
