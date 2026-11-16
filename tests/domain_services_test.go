package tests

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
	"perdimidocumento/src/domain/services"
)

type repositoryStub struct {
	createUserFn          func(context.Context, string, string) error
	userByNameFn          func(context.Context, string) (models.User, error)
	createSessionFn       func(context.Context, int64, string, time.Time) error
	userBySessionFn       func(context.Context, string) (models.User, error)
	deleteSessionFn       func(context.Context, string) error
	registerDocumentFn    func(context.Context, models.Document) (models.Document, error)
	logRejectedDocumentFn func(context.Context, int64, string) error
	searchDocumentsFn     func(context.Context, string) ([]models.Document, error)
	subscribeFn           func(context.Context, int64, string) error
	notificationsFn       func(context.Context, int64) ([]models.Notification, error)
}

func (r repositoryStub) CreateUser(ctx context.Context, username, passwordHash string) error {
	if r.createUserFn != nil {
		return r.createUserFn(ctx, username, passwordHash)
	}
	return nil
}

func (r repositoryStub) UserByName(ctx context.Context, username string) (models.User, error) {
	if r.userByNameFn != nil {
		return r.userByNameFn(ctx, username)
	}
	return models.User{}, nil
}

func (r repositoryStub) CreateSession(ctx context.Context, userID int64, hash string, expiry time.Time) error {
	if r.createSessionFn != nil {
		return r.createSessionFn(ctx, userID, hash, expiry)
	}
	return nil
}

func (r repositoryStub) UserBySession(ctx context.Context, hash string) (models.User, error) {
	if r.userBySessionFn != nil {
		return r.userBySessionFn(ctx, hash)
	}
	return models.User{}, nil
}

func (r repositoryStub) DeleteSession(ctx context.Context, hash string) error {
	if r.deleteSessionFn != nil {
		return r.deleteSessionFn(ctx, hash)
	}
	return nil
}

func (r repositoryStub) RegisterDocument(ctx context.Context, doc models.Document) (models.Document, error) {
	if r.registerDocumentFn != nil {
		return r.registerDocumentFn(ctx, doc)
	}
	return doc, nil
}

func (r repositoryStub) LogRejectedDocument(ctx context.Context, userID int64, number string) error {
	if r.logRejectedDocumentFn != nil {
		return r.logRejectedDocumentFn(ctx, userID, number)
	}
	return nil
}

func (r repositoryStub) SearchDocuments(ctx context.Context, number string) ([]models.Document, error) {
	if r.searchDocumentsFn != nil {
		return r.searchDocumentsFn(ctx, number)
	}
	return nil, nil
}

func (r repositoryStub) Subscribe(ctx context.Context, userID int64, number string) error {
	if r.subscribeFn != nil {
		return r.subscribeFn(ctx, userID, number)
	}
	return nil
}

func (r repositoryStub) Notifications(ctx context.Context, userID int64) ([]models.Notification, error) {
	if r.notificationsFn != nil {
		return r.notificationsFn(ctx, userID)
	}
	return nil, nil
}

type passwordHasherStub struct {
	hashFn    func(string) (string, error)
	compareFn func(string, string) error
}

func (h passwordHasherStub) Hash(password string) (string, error) {
	if h.hashFn != nil {
		return h.hashFn(password)
	}
	return "", nil
}

func (h passwordHasherStub) Compare(hash, password string) error {
	if h.compareFn != nil {
		return h.compareFn(hash, password)
	}
	return nil
}

type tokenProviderStub struct {
	newFn  func() (string, error)
	hashFn func(string) string
}

func (p tokenProviderStub) New() (string, error) {
	if p.newFn != nil {
		return p.newFn()
	}
	return "", nil
}

func (p tokenProviderStub) Hash(token string) string {
	if p.hashFn != nil {
		return p.hashFn(token)
	}
	return token
}

func TestAuthServiceRegisterUsesHasherAndRepository(t *testing.T) {
	ctx := context.Background()
	calledHash := ""
	calledUser := ""
	calledStoredHash := ""

	service := services.AuthService{
		Hasher: passwordHasherStub{hashFn: func(password string) (string, error) {
			calledHash = password
			return "hash-seguro", nil
		}},
		Repository: repositoryStub{createUserFn: func(_ context.Context, username, passwordHash string) error {
			calledUser = username
			calledStoredHash = passwordHash
			return nil
		}},
	}

	if err := service.Register(ctx, "ana", "clave123"); err != nil {
		t.Fatalf("registro inesperado: %v", err)
	}
	if calledHash != "clave123" {
		t.Fatalf("clave enviada al hasher: %q", calledHash)
	}
	if calledUser != "ana" || calledStoredHash != "hash-seguro" {
		t.Fatalf("persistencia inesperada: usuario=%q hash=%q", calledUser, calledStoredHash)
	}
}

func TestAuthServiceLoginInvalidCredentials(t *testing.T) {
	t.Run("usuario no existe", func(t *testing.T) {
		service := services.AuthService{
			Repository: repositoryStub{userByNameFn: func(context.Context, string) (models.User, error) {
				return models.User{}, ports.ErrNotFound
			}},
			Hasher: passwordHasherStub{},
			Tokens: tokenProviderStub{},
		}
		_, err := service.Login(context.Background(), "ana", "clave")
		if !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Fatalf("error esperado: %v", err)
		}
	})

	t.Run("clave incorrecta", func(t *testing.T) {
		service := services.AuthService{
			Repository: repositoryStub{userByNameFn: func(context.Context, string) (models.User, error) {
				return models.User{ID: 1, PasswordHash: "hash"}, nil
			}},
			Hasher: passwordHasherStub{compareFn: func(string, string) error {
				return errors.New("invalid")
			}},
			Tokens: tokenProviderStub{},
		}
		_, err := service.Login(context.Background(), "ana", "mala")
		if !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Fatalf("error esperado: %v", err)
		}
	})
}

func TestAuthServiceLoginSuccessStoresHashedTokenAndTTL(t *testing.T) {
	ctx := context.Background()
	var savedUserID int64
	var savedHash string
	var savedExpiry time.Time

	service := services.AuthService{
		Repository: repositoryStub{
			userByNameFn: func(context.Context, string) (models.User, error) {
				return models.User{ID: 11, Username: "ana", PasswordHash: "hash"}, nil
			},
			createSessionFn: func(_ context.Context, userID int64, hash string, expiry time.Time) error {
				savedUserID = userID
				savedHash = hash
				savedExpiry = expiry
				return nil
			},
		},
		Hasher: passwordHasherStub{compareFn: func(string, string) error { return nil }},
		Tokens: tokenProviderStub{
			newFn:  func() (string, error) { return "token-claro", nil },
			hashFn: func(token string) string { return "hash:" + token },
		},
		SessionTTL: 90 * time.Minute,
	}

	before := time.Now()
	session, err := service.Login(ctx, "ana", "ok")
	after := time.Now()
	if err != nil {
		t.Fatalf("login inesperado: %v", err)
	}
	if session.Token != "token-claro" || session.User.Username != "ana" {
		t.Fatalf("respuesta inesperada: %+v", session)
	}
	if savedUserID != 11 || savedHash != "hash:token-claro" {
		t.Fatalf("sesión persistida inesperada: user=%d hash=%q", savedUserID, savedHash)
	}
	min := before.Add(90*time.Minute - 10*time.Second)
	max := after.Add(90*time.Minute + 10*time.Second)
	if savedExpiry.Before(min) || savedExpiry.After(max) {
		t.Fatalf("expiración fuera de rango: %v", savedExpiry.Sub(before))
	}
}

func TestAuthServiceLoginUsesDefaultTTL(t *testing.T) {
	service := services.AuthService{
		Repository: repositoryStub{
			userByNameFn: func(context.Context, string) (models.User, error) {
				return models.User{ID: 2, PasswordHash: "hash"}, nil
			},
			createSessionFn: func(_ context.Context, _ int64, _ string, expiry time.Time) error {
				if expiry.IsZero() {
					t.Fatalf("expiry no debe ser cero")
				}
				return nil
			},
		},
		Hasher: passwordHasherStub{compareFn: func(string, string) error { return nil }},
		Tokens: tokenProviderStub{newFn: func() (string, error) { return "t", nil }},
	}

	before := time.Now()
	session, err := service.Login(context.Background(), "ana", "ok")
	after := time.Now()
	if err != nil {
		t.Fatalf("login inesperado: %v", err)
	}
	min := before.Add(23*time.Hour + 50*time.Minute)
	max := after.Add(24*time.Hour + 10*time.Minute)
	if session.ExpiresAt.Before(min) || session.ExpiresAt.After(max) {
		t.Fatalf("TTL por defecto fuera de rango: %v", session.ExpiresAt.Sub(before))
	}
}

func TestAuthServiceHashesTokenForUserLookupAndLogout(t *testing.T) {
	ctx := context.Background()
	hashes := []string{}
	service := services.AuthService{
		Repository: repositoryStub{
			userBySessionFn: func(_ context.Context, hash string) (models.User, error) {
				hashes = append(hashes, hash)
				return models.User{ID: 7, Username: "ana"}, nil
			},
			deleteSessionFn: func(_ context.Context, hash string) error {
				hashes = append(hashes, hash)
				return nil
			},
		},
		Tokens: tokenProviderStub{hashFn: func(token string) string { return "hash:" + token }},
	}

	user, err := service.UserByToken(ctx, "abc")
	if err != nil {
		t.Fatalf("búsqueda de sesión inesperada: %v", err)
	}
	if user.ID != 7 {
		t.Fatalf("usuario inesperado: %+v", user)
	}
	if err := service.Logout(ctx, "abc"); err != nil {
		t.Fatalf("logout inesperado: %v", err)
	}
	if len(hashes) != 2 || hashes[0] != "hash:abc" || hashes[1] != "hash:abc" {
		t.Fatalf("se esperaba hash del token en ambos llamados: %+v", hashes)
	}
}

func TestDocumentProxyRegisterSuccessAndAuditOnReject(t *testing.T) {
	t.Run("registro exitoso", func(t *testing.T) {
		captured := models.Document{}
		proxy := services.DocumentProxy{
			Factory: models.DocumentFactory{},
			Repository: repositoryStub{registerDocumentFn: func(_ context.Context, doc models.Document) (models.Document, error) {
				captured = doc
				doc.ID = 88
				return doc, nil
			}},
		}

		doc, err := proxy.Register(context.Background(), 9, " 001 ", " Cédula ", " Ana ", " Oficina ")
		if err != nil {
			t.Fatalf("registro inesperado: %v", err)
		}
		if captured.Number != "001" || captured.Type != "Cédula" || captured.OwnerName != "Ana" || captured.Address != "Oficina" || captured.ReportedBy != 9 {
			t.Fatalf("documento enviado al repositorio inesperado: %+v", captured)
		}
		if doc.ID != 88 {
			t.Fatalf("respuesta del repositorio no propagada: %+v", doc)
		}
	})

	t.Run("rechazo audita número truncado", func(t *testing.T) {
		loggedUser := int64(0)
		loggedNumber := ""
		proxy := services.DocumentProxy{
			Factory: models.DocumentFactory{},
			Repository: repositoryStub{logRejectedDocumentFn: func(_ context.Context, userID int64, number string) error {
				loggedUser = userID
				loggedNumber = number
				return nil
			}},
		}

		number := "   " + strings.Repeat("9", 205) + "  "
		_, err := proxy.Register(context.Background(), 23, number, "", "Ana", "Oficina")
		if !errors.Is(err, models.ErrInvalidDocument) {
			t.Fatalf("se esperaba error de validación: %v", err)
		}
		if loggedUser != 23 {
			t.Fatalf("usuario auditado inesperado: %d", loggedUser)
		}
		if loggedNumber != strings.Repeat("9", 200) {
			t.Fatalf("número auditado inesperado (len=%d): %q", len(loggedNumber), loggedNumber)
		}
	})
}

func TestReaderServicesDelegateToRepository(t *testing.T) {
	ctx := context.Background()
	docs := []models.Document{{ID: 1, Number: "123"}}
	notes := []models.Notification{{ID: 7, Document: models.Document{ID: 4, Number: "123"}}}
	calledSearch := ""
	calledSubscribeUser := int64(0)
	calledSubscribeNumber := ""
	calledNotificationsUser := int64(0)

	repo := repositoryStub{
		searchDocumentsFn: func(_ context.Context, number string) ([]models.Document, error) {
			calledSearch = number
			return docs, nil
		},
		subscribeFn: func(_ context.Context, userID int64, number string) error {
			calledSubscribeUser = userID
			calledSubscribeNumber = number
			return nil
		},
		notificationsFn: func(_ context.Context, userID int64) ([]models.Notification, error) {
			calledNotificationsUser = userID
			return notes, nil
		},
	}

	search := services.DocumentSearchService{Repository: repo}
	gotDocs, err := search.Search(ctx, "001")
	if err != nil || len(gotDocs) != 1 || gotDocs[0].ID != 1 || calledSearch != "001" {
		t.Fatalf("búsqueda inesperada: docs=%+v err=%v called=%q", gotDocs, err, calledSearch)
	}

	subscriptions := services.SubscriptionService{Repository: repo}
	if err := subscriptions.Subscribe(ctx, 55, "001"); err != nil {
		t.Fatalf("suscripción inesperada: %v", err)
	}
	if calledSubscribeUser != 55 || calledSubscribeNumber != "001" {
		t.Fatalf("suscripción delegada inesperada: user=%d number=%q", calledSubscribeUser, calledSubscribeNumber)
	}

	notificationService := services.NotificationService{Repository: repo}
	gotNotes, err := notificationService.Notifications(ctx, 55)
	if err != nil || len(gotNotes) != 1 || gotNotes[0].ID != 7 || calledNotificationsUser != 55 {
		t.Fatalf("notificaciones inesperadas: %+v err=%v user=%d", gotNotes, err, calledNotificationsUser)
	}
}
