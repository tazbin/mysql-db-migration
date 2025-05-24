SET ?= default-set

migration-up:
	go run main.go do-migrate $(SET)

migration-down:
	go run main.go undo-migrate $(SET)
