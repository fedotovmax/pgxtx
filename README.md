# Download

go get -u github.com/fedotovmax/pgxtx@{version}

# How to use

```go
func main() {
	dsn := "postgres://user:pass123@localhost:5432/baza1?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	} else {
		log.Println("DB Connected successfully!")
	}

	manager, err := pgxtx.Init(pool)
	if err != nil {
		log.Fatal(err)
	}

	ex := manager.GetExtractor()

	repo := repository.NewUserRepo(ex)

	wrapCtx := context.Background()

	err = manager.Wrap(wrapCtx, func(ctx context.Context) error {
		id, err := repo.CreateUser(ctx, "Alexander")
		if err != nil {
			return fmt.Errorf("repository.user.CreateUser: %w", err)
		}

		u, err := repo.GetUser(ctx, *id)
		if err != nil {
			return fmt.Errorf("repository.user.GetUser: %w", err)
		}

		log.Printf("User: %v", u)

		dto := domain.User{
			ID:   u.ID,
			Name: "Ivan",
		}

		err = repo.UpdateUser(ctx, dto)
		if err != nil {
			return fmt.Errorf("repository.user.UpdateUser: %w", err)
		}

		u, err = repo.GetUser(ctx, dto.ID)
		if err != nil {
			return fmt.Errorf("repository.user.GetUser: %w", err)
		}

		log.Printf("User after update: %v", u)

		// Uncomment to simulate error
		// return fmt.Errorf("unexpected error in FN")

		return nil
	})

	if err != nil {
		log.Fatalf("Wrap: %v", err)
	}

	log.Println("Transaction successfully executed!")
}
```
