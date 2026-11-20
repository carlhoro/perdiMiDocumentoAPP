package user

// User representa una cuenta local identificada por su ID.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
}
