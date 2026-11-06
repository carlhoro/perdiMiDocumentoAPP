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
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
	"perdimidocumento/src/domain/services"
	in "perdimidocumento/src/infrastructure/in"
	out "perdimidocumento/src/infrastructure/out/db"
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

type documentResponse struct {
	ID        int64  `json:"id"`
	Number    string `json:"numero"`
	Type      string `json:"tipo"`
	OwnerName string `json:"nombre_titular"`
	Address   string `json:"direccion"`
	CreatedAt string `json:"fecha"`
}

type notificationResponse struct {
	ID        int64            `json:"id"`
	Document  documentResponse `json:"documento"`
	CreatedAt string           `json:"fecha"`
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

	var docs []documentResponse
	request(t, ana, "GET", server.URL+"/api/documentos?numero=00123", "", http.StatusOK, &docs)
	if len(docs) != 1 || docs[0].Address != "Oficina" || docs[0].Number != "00123" {
		t.Fatalf("documentos: %+v", docs)
	}
	var notes []notificationResponse
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

func TestHTTPDocumentAndNotificationContract(t *testing.T) {
	_, server := setup(t)
	c := client(t)
	login(t, c, server.URL, "ana")

	for _, path := range []string{"/api/documentos?numero=00123", "/api/notificaciones"} {
		var raw json.RawMessage
		request(t, c, "GET", server.URL+path, "", http.StatusOK, &raw)
		if string(raw) != "[]" {
			t.Fatalf("%s: lista vacía = %s; se esperaba []", path, raw)
		}
	}

	request(t, c, "POST", server.URL+"/api/suscripciones", `{"numero":"00123"}`, http.StatusOK, nil)
	var document map[string]any
	request(t, c, "POST", server.URL+"/api/documentos", `{"numero":"00123","tipo":"Cédula","nombre_titular":"Ana","direccion":"Oficina"}`, http.StatusCreated, &document)
	if len(document) != 6 || document["id"] != float64(1) || document["numero"] != "00123" ||
		document["tipo"] != "Cédula" || document["nombre_titular"] != "Ana" || document["direccion"] != "Oficina" {
		t.Fatalf("contrato de documento: %#v", document)
	}
	date, ok := document["fecha"].(string)
	if !ok {
		t.Fatalf("fecha de documento: %#v", document["fecha"])
	}
	if _, err := time.Parse(time.RFC3339, date); err != nil {
		t.Fatalf("fecha de documento inválida: %v", err)
	}

	var documents []map[string]any
	request(t, c, "GET", server.URL+"/api/documentos?numero=00123", "", http.StatusOK, &documents)
	if len(documents) != 1 || !reflect.DeepEqual(documents[0], document) {
		t.Fatalf("documentos: %#v; creado: %#v", documents, document)
	}
	var notifications []map[string]any
	request(t, c, "GET", server.URL+"/api/notificaciones", "", http.StatusOK, &notifications)
	if len(notifications) != 1 || len(notifications[0]) != 3 || notifications[0]["id"] != float64(1) ||
		!reflect.DeepEqual(notifications[0]["documento"], document) {
		t.Fatalf("contrato de notificaciones: %#v", notifications)
	}
	notificationDate, ok := notifications[0]["fecha"].(string)
	if !ok {
		t.Fatalf("fecha de notificación: %#v", notifications[0]["fecha"])
	}
	if _, err := time.Parse(time.RFC3339, notificationDate); err != nil {
		t.Fatalf("fecha de notificación inválida: %v", err)
	}
}

func TestSQLiteDomainMappings(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if err := repo.CreateUser(ctx, "ana", "hash"); err != nil {
		t.Fatal(err)
	}
	user, err := repo.UserByName(ctx, "ana")
	if err != nil || user.ID == 0 || user.Username != "ana" || user.PasswordHash != "hash" {
		t.Fatalf("usuario: %+v, error: %v", user, err)
	}
	if err := repo.CreateSession(ctx, user.ID, "token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	sessionUser, err := repo.UserBySession(ctx, "token")
	if err != nil || sessionUser.ID != user.ID || sessionUser.Username != user.Username || sessionUser.PasswordHash != "" {
		t.Fatalf("usuario de sesión: %+v, error: %v", sessionUser, err)
	}
	if _, err := repo.UserByName(ctx, "ausente"); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("usuario ausente: %v", err)
	}
	if _, err := repo.UserBySession(ctx, "ausente"); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("sesión ausente: %v", err)
	}
	if err := repo.Subscribe(ctx, user.ID, "00123"); err != nil {
		t.Fatal(err)
	}
	input := models.Document{
		ID: 999, Number: "00123", Type: "Cédula", OwnerName: "Ana", Address: "Oficina",
		ReportedBy: user.ID, CreatedAt: time.Unix(42, 0).UTC(),
	}
	doc, err := repo.RegisterDocument(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID == 0 || doc.ID == input.ID || doc.Number != input.Number || doc.Type != input.Type ||
		doc.OwnerName != input.OwnerName || doc.Address != input.Address || doc.ReportedBy != input.ReportedBy ||
		doc.CreatedAt.Equal(input.CreatedAt) || doc.CreatedAt.Location() != time.UTC {
		t.Fatalf("documento registrado: %+v; entrada: %+v", doc, input)
	}
	var persistedID, persistedUser, persistedDate int64
	var number, kind, owner, address string
	if err := repo.DB.QueryRowContext(ctx, `SELECT id,number,type,owner_name,address,reported_by,created_at FROM documents WHERE id=?`, doc.ID).
		Scan(&persistedID, &number, &kind, &owner, &address, &persistedUser, &persistedDate); err != nil {
		t.Fatal(err)
	}
	if persistedID != doc.ID || number != doc.Number || kind != doc.Type || owner != doc.OwnerName ||
		address != doc.Address || persistedUser != doc.ReportedBy || persistedDate != doc.CreatedAt.Unix() {
		t.Fatalf("fila de documento no coincide con el dominio: %+v", doc)
	}
	documents, err := repo.SearchDocuments(ctx, doc.Number)
	if err != nil || len(documents) != 1 {
		t.Fatalf("documentos: %+v, error: %v", documents, err)
	}
	expected := doc
	expected.ReportedBy = 0 // La consulta de búsqueda no selecciona este campo.
	if !reflect.DeepEqual(documents[0], expected) || documents[0].CreatedAt.Location() != time.UTC {
		t.Fatalf("documento mapeado: %+v; esperado: %+v", documents[0], expected)
	}
	notifications, err := repo.Notifications(ctx, user.ID)
	if err != nil || len(notifications) != 1 {
		t.Fatalf("notificaciones: %+v, error: %v", notifications, err)
	}
	if !reflect.DeepEqual(notifications[0].Document, expected) || notifications[0].CreatedAt.IsZero() ||
		notifications[0].CreatedAt.Location() != time.UTC {
		t.Fatalf("notificación mapeada: %+v", notifications[0])
	}
}
