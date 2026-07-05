// Command seed-admin creates the first admin user in an environment where the
// automatic dev seed does not run (production). It reuses the user service, so
// the admin is created through the exact same validation and hashing path as
// any other user — no manual bcrypt hashes, no duplicated rules.
//
// Usage (run after migrations, with DB_* env vars pointing at the target DB):
//
//	go run ./cmd/seed-admin -name "Admin" -email admin@finishline.com -password 's3cret...'
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"

	"finish-line/internal/common/config"
	"finish-line/internal/common/postgres"
	"finish-line/internal/common/security"
	userpostgres "finish-line/internal/user/adapters/postgres"
	"finish-line/internal/user/domain"
	userservice "finish-line/internal/user/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed-admin:", err)
		os.Exit(1)
	}
}

func run() error {
	name := flag.String("name", "", "admin display name")
	email := flag.String("email", "", "admin email (login)")
	password := flag.String("password", "", "admin password (min 8 chars)")
	flag.Parse()

	if *name == "" || *email == "" || *password == "" {
		flag.Usage()
		return errors.New("-name, -email and -password are all required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := postgres.Connect(cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	// Same wiring as the app, minus the session revoker: seeding only calls
	// Register, which never revokes sessions.
	svc := userservice.New(userpostgres.NewRepository(db), security.NewBcryptHasher(), noopRevoker{})

	_, err = svc.Register(context.Background(), *name, *email, *password)
	switch {
	case err == nil:
		fmt.Printf("created admin %q\n", *email)
		return nil
	case errors.Is(err, domain.ErrEmailTaken):
		fmt.Printf("admin %q already exists — nothing to do\n", *email)
		return nil
	default:
		return fmt.Errorf("creating admin: %w", err)
	}
}

// noopRevoker satisfies the user service's SessionRevoker dependency. The seed
// path only calls Register, so RevokeAllSessions is never invoked.
type noopRevoker struct{}

func (noopRevoker) RevokeAllSessions(context.Context, uuid.UUID) error { return nil }
