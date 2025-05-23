migration-up:
	go run main.go do-migrate

migration-down:
	go run main.go undo-migrate