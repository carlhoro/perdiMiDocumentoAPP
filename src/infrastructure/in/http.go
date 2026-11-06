package in

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
)

//go:embed index.html
var static embed.FS

type Server struct {
	Repository    ports.Repository
	Registrar     ports.DocumentRegistrar
	SecureCookies bool
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "index.html")
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"estado": "ok"})
	})
	mux.HandleFunc("POST /api/registro", s.register)
	mux.HandleFunc("POST /api/ingreso", s.login)
	mux.HandleFunc("POST /api/salir", s.auth(s.logout))
	mux.HandleFunc("GET /api/sesion", s.auth(s.session))
	mux.HandleFunc("POST /api/documentos", s.auth(s.createDocument))
	mux.HandleFunc("GET /api/documentos", s.auth(s.searchDocuments))
	mux.HandleFunc("POST /api/suscripciones", s.auth(s.subscribe))
	mux.HandleFunc("GET /api/notificaciones", s.auth(s.notifications))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("respuesta HTTP: %v", err)
	}
}

func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func internal(w http.ResponseWriter, err error) {
	log.Printf("error interno: %v", err)
	fail(w, http.StatusInternalServerError, "Error interno; inténtelo de nuevo")
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		fail(w, http.StatusUnsupportedMediaType, "Se requiere application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		fail(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		fail(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

type credentials struct {
	Username string `json:"usuario"`
	Password string `json:"clave"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if c.Username == "" || utf8.RuneCountInString(c.Username) > 100 || len(c.Password) < 8 || len(c.Password) > 72 {
		fail(w, http.StatusBadRequest, "Usuario obligatorio (máximo 100 caracteres) y clave de 8 a 72 bytes")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		internal(w, err)
		return
	}
	err = s.Repository.CreateUser(r.Context(), c.Username, string(hash))
	if errors.Is(err, ports.ErrConflict) {
		fail(w, http.StatusConflict, "El usuario ya existe")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"mensaje": "Usuario creado; inicie sesión"})
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	if len(c.Password) > 72 {
		fail(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}
	u, err := s.Repository.UserByName(r.Context(), strings.TrimSpace(c.Username))
	if errors.Is(err, ports.ErrNotFound) {
		fail(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(c.Password)) != nil {
		fail(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		internal(w, err)
		return
	}
	token := hex.EncodeToString(bytes)
	expires := time.Now().Add(24 * time.Hour)
	if err := s.Repository.CreateSession(r.Context(), u.ID, tokenHash(token), expires); err != nil {
		internal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "sesion", Value: token, Path: "/", HttpOnly: true, Secure: s.SecureCookies, SameSite: http.SameSiteStrictMode, Expires: expires})
	writeJSON(w, http.StatusOK, map[string]string{"usuario": u.Username})
}

type authenticatedHandler func(http.ResponseWriter, *http.Request, models.User)

func (s *Server) auth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sesion")
		if err != nil || len(cookie.Value) != 64 {
			fail(w, http.StatusUnauthorized, "Inicie sesión")
			return
		}
		u, err := s.Repository.UserBySession(r.Context(), tokenHash(cookie.Value))
		if errors.Is(err, ports.ErrNotFound) {
			fail(w, http.StatusUnauthorized, "Inicie sesión")
			return
		}
		if err != nil {
			internal(w, err)
			return
		}
		next(w, r, u)
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, _ models.User) {
	cookie, _ := r.Cookie("sesion")
	if err := s.Repository.DeleteSession(r.Context(), tokenHash(cookie.Value)); err != nil {
		internal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "sesion", Path: "/", HttpOnly: true, Secure: s.SecureCookies, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]string{"mensaje": "Sesión cerrada"})
}

func (s *Server) session(w http.ResponseWriter, _ *http.Request, u models.User) {
	writeJSON(w, http.StatusOK, map[string]string{"usuario": u.Username})
}

func (s *Server) createDocument(w http.ResponseWriter, r *http.Request, u models.User) {
	var input struct {
		Number    string `json:"numero"`
		Type      string `json:"tipo"`
		OwnerName string `json:"nombre_titular"`
		Address   string `json:"direccion"`
	}
	if !decode(w, r, &input) {
		return
	}
	doc, err := s.Registrar.Register(r.Context(), u.ID, input.Number, input.Type, input.OwnerName, input.Address)
	if errors.Is(err, models.ErrInvalidDocument) {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, documentToResponse(doc))
}

func validNumber(number string) bool { return number != "" && utf8.RuneCountInString(number) <= 200 }

func (s *Server) searchDocuments(w http.ResponseWriter, r *http.Request, _ models.User) {
	number := strings.TrimSpace(r.URL.Query().Get("numero"))
	if !validNumber(number) {
		fail(w, http.StatusBadRequest, "Número obligatorio (máximo 200 caracteres)")
		return
	}
	docs, err := s.Repository.SearchDocuments(r.Context(), number)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, documentsToResponse(docs))
}

func (s *Server) subscribe(w http.ResponseWriter, r *http.Request, u models.User) {
	var input struct {
		Number string `json:"numero"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.Number = strings.TrimSpace(input.Number)
	if !validNumber(input.Number) {
		fail(w, http.StatusBadRequest, "Número obligatorio (máximo 200 caracteres)")
		return
	}
	if err := s.Repository.Subscribe(r.Context(), u.ID, input.Number); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"mensaje": "Suscripción activa"})
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request, u models.User) {
	notifications, err := s.Repository.Notifications(r.Context(), u.ID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationsToResponse(notifications))
}
