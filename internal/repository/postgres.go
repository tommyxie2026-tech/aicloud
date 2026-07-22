package repository

// PostgreSQLRepository is a persistence seam for v0.1. Runtime persistence
// currently uses memory repositories so the API starts without dependencies.
// db/migrations contains the first PostgreSQL schema contract.
type PostgreSQLRepository struct{ DSN string }

func NewPostgreSQLRepository(dsn string) *PostgreSQLRepository {
	return &PostgreSQLRepository{DSN: dsn}
}
