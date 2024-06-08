include configs/.env

lint:
	golangci-lint run --issues-exit-code=0

MIGRATIONS_LOCALE="internal/adapters/repository/migrations"
CDB_DSN="cockroachdb://${CDB_USER}@${CDB_HOST}:${CDB_PORT}/${CDB_DATABASE}?sslmode=disable"
migration-up: migrate-check-deps
	migrate -path ${MIGRATIONS_LOCALE} -database ${CDB_DSN} up

create-migration: migrate-check-deps
	migrate create -ext sql -dir ${MIGRATIONS_LOCALE} -seq $(name)

migration-down: migrate-check-deps
	migrate -database ${CDB_DSN} \
		-path ${MIGRATIONS_LOCALE} down 1

migrate-check-deps:
	@if [ -z `which migrate` ]; then \
		echo "[go get] installing golang-migrate cmd with cockroachdb support";\
		if [ "${GO111MODULE}" = "off" ]; then \
			echo "[go get] installing github.com/golang-migrate/migrate/cmd/migrate"; \
			go get -tags 'cockroachdb postgres' -u github.com/golang-migrate/migrate/cmd/migrate;\
			go install -tags 'cockroachdb postgres' github.com/golang-migrate/migrate/cmd/migrate;\
		else \
			echo "[go get] installing github.com/golang-migrate/migrate/v4/cmd/migrate"; \
			go get -tags 'cockroachdb postgres' -u github.com/golang-migrate/migrate/v4/cmd/migrate;\
			go install -tags 'cockroachdb postgres' github.com/golang-migrate/migrate/v4/cmd/migrate;\
		fi \
	fi
