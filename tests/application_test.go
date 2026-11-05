package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/services"
	in "perdimidocumento/src/infrastructure/in"
	"perdimidocumento/src/infrastructure/out"
)

func setup(t *testing.T) (*out.SQLiteRepository, *httptest.Server) {
	t.Helper()
	db, err := out.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := &out.SQLiteRepository{DB: db}
	server := httptest.NewServer((&in.Server{Repository: repo, Registrar: services.DocumentProxy{Repository: repo, Factory: models.DocumentFactory{}}}).Handler())
	t.Cleanup(server.Close)
	return repo, server
}

func client(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func request(t *testing.T, c *http.Client, method, url, body string, status int, result any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != status {
		t.Fatalf("%s %s: estado %d, esperado %d: %s", method, url, resp.StatusCode, status, data)
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			t.Fatalf("JSON: %v: %s", err, data)
		}
	}
}

func login(t *testing.T, c *http.Client, base, user string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"usuario": user, "clave": "contraseña123"})
	if err != nil {
		t.Fatal(err)
	}
	request(t, c, "POST", base+"/api/registro", string(body), http.StatusCreated, nil)
	request(t, c, "POST", base+"/api/ingreso", string(body), http.StatusOK, nil)
}

func TestApplicationFlow(t *testing.T) {
	repo, server := setup(t)
	ana, reporter, other, second := client(t), client(t), client(t), client(t)
	request(t, ana, "GET", server.URL+"/api/notificaciones", "", http.StatusUnauthorized, nil)
	login(t, ana, server.URL, "ana")
	login(t, reporter, server.URL, "reportero")
	login(t, other, server.URL, "otro")
	login(t, second, server.URL, "segunda")

	request(t, ana, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	request(t, ana, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	request(t, second, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	request(t, reporter, "POST", server.URL+"/api/documentos", `{"numero":"00123","tipo":"","nombre_titular":"Ana","direccion":"Oficina"}`, http.StatusBadRequest, nil)
	request(t, reporter, "POST", server.URL+"/api/documentos", `{"numero":" 00123 ","tipo":"Cédula","nombre_titular":"Ana","direccion":"Oficina"}`, http.StatusCreated, nil)

	var docs []models.Document
	request(t, ana, "GET", server.URL+"/api/documentos?numero=00123", "", http.StatusOK, &docs)
	if len(docs) != 1 || docs[0].Address != "Oficina" || docs[0].Number != "00123" {
		t.Fatalf("documentos: %+v", docs)
	}
	var notes []models.Notification
	request(t, ana, "GET", server.URL+"/api/notificaciones", "", http.StatusOK, &notes)
	if len(notes) != 1 || notes[0].Document.ID != docs[0].ID {
		t.Fatalf("notificaciones: %+v", notes)
	}
	request(t, second, "GET", server.URL+"/api/notificaciones", "", http.StatusOK, &notes)
	if len(notes) != 1 || notes[0].Document.ID != docs[0].ID {
		t.Fatalf("segundo observador: %+v", notes)
	}
	request(t, other, "GET", server.URL+"/api/notificaciones", "", http.StatusOK, &notes)
	if len(notes) != 0 {
		t.Fatalf("otro usuario recibió notificaciones: %+v", notes)
	}
	request(t, other, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	request(t, other, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	request(t, other, "GET", server.URL+"/api/notificaciones", "", http.StatusOK, &notes)
	if len(notes) != 1 {
		t.Fatalf("suscripción posterior: %+v", notes)
	}

	var success, rejected int
	if err := repo.DB.QueryRow(`SELECT COUNT(*) FROM logs l JOIN users u ON u.id=l.user_id WHERE l.operation='registro' AND l.document_number='00123' AND l.document_id IS NOT NULL AND l.created_at>0 AND u.username='reportero'`).Scan(&success); err != nil {
		t.Fatal(err)
	}
	if err := repo.DB.QueryRow(`SELECT COUNT(*) FROM logs WHERE operation='registro_rechazado' AND document_number='00123' AND document_id IS NULL`).Scan(&rejected); err != nil {
		t.Fatal(err)
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("logs: registro=%d rechazo=%d", success, rejected)
	}

	request(t, ana, "POST", server.URL+"/api/salir", `{}`, http.StatusOK, nil)
	request(t, ana, "GET", server.URL+"/api/notificaciones", "", http.StatusUnauthorized, nil)
}

func TestAtomicRegistration(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if err := repo.CreateUser(ctx, "ana", "hash"); err != nil {
		t.Fatal(err)
	}
	u, err := repo.UserByName(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Subscribe(ctx, u.ID, "123"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DB.Exec(`CREATE TRIGGER reject_notifications BEFORE INSERT ON notifications BEGIN SELECT RAISE(FAIL, 'test'); END`); err != nil {
		t.Fatal(err)
	}
	doc, err := (models.DocumentFactory{}).Create("123", "id", "Ana", "Oficina", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RegisterDocument(ctx, doc); err == nil {
		t.Fatal("se esperaba un error de notificación")
	}
	for _, table := range []string{"documents", "logs", "notifications"} {
		var count int
		if err := repo.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s no se revirtió: %d", table, count)
		}
	}
}

func TestConcurrentSubscribeAndRegister(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if err := repo.CreateUser(ctx, "ana", "hash"); err != nil {
		t.Fatal(err)
	}
	u, err := repo.UserByName(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := (models.DocumentFactory{}).Create("123", "id", "Ana", "Oficina", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); errs <- repo.Subscribe(ctx, u.ID, doc.Number) }()
	go func() { defer wg.Done(); _, err := repo.RegisterDocument(ctx, doc); errs <- err }()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	notes, err := repo.Notifications(ctx, u.ID)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notificaciones: %+v, error: %v", notes, err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := out.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := &out.SQLiteRepository{DB: db}
	ctx := context.Background()
	if err := repo.CreateUser(ctx, "ana", "hash"); err != nil {
		t.Fatal(err)
	}
	u, err := repo.UserByName(ctx, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Subscribe(ctx, u.ID, "123"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = out.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo.DB = db
	doc, err := (models.DocumentFactory{}).Create("123", "id", "Ana", "Oficina", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RegisterDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	notes, err := repo.Notifications(ctx, u.ID)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notificaciones: %+v, error: %v", notes, err)
	}
}

func TestRejectedAuditFailureIsNotHidden(t *testing.T) {
	repo, _ := setup(t)
	proxy := services.DocumentProxy{Repository: repo, Factory: models.DocumentFactory{}}
	_, err := proxy.Register(context.Background(), 999, "123", "", "Ana", "Oficina")
	if err == nil || !strings.Contains(err.Error(), "auditar") {
		t.Fatalf("error esperado de auditoría: %v", err)
	}
	if errors.Is(err, models.ErrInvalidDocument) {
		t.Fatalf("el error de auditoría no debe presentarse como validación: %v", err)
	}
}

func TestLoginAndValidation(t *testing.T) {
	_, server := setup(t)
	c := client(t)
	request(t, c, "POST", server.URL+"/api/registro", `{"usuario":"ana","clave":"corta"}`, http.StatusBadRequest, nil)
	login(t, c, server.URL, "ana")
	request(t, c, "POST", server.URL+"/api/registro", `{"usuario":"ana","clave":"contraseña123"}`, http.StatusConflict, nil)
	request(t, c, "POST", server.URL+"/api/ingreso", `{"usuario":"ana","clave":"incorrecta"}`, http.StatusUnauthorized, nil)
	request(t, c, "GET", server.URL+"/api/documentos?numero=%20", "", http.StatusBadRequest, nil)
	request(t, c, "POST", server.URL+"/api/suscripciones", `{"numero":"123","extra":"x"}`, http.StatusBadRequest, nil)
	req, err := http.NewRequest("POST", server.URL+"/api/registro", bytes.NewBufferString(`{"usuario":"x","clave":"abcdefgh"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("content type: %d", resp.StatusCode)
	}
}
