include configs/.env

lint:
	golangci-lint run --issues-exit-code=0

MIGRATIONS_LOCALE="internal/adapters/repository/cdb/migrations"
migration-up: migrate-check-deps check-cdb-env
	migrate -path ${MIGRATIONS_LOCALE} -database ${CDB_DSN} up

create-migration: migrate-check-deps check-cdb-env
	migrate create -ext sql -dir ${MIGRATIONS_LOCALE} -seq $(name)

migration-down: migrate-check-deps check-cdb-env
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

define dsn_missing_error

CDB_DSN envvar is undefined. To run the migrations this envvar
must point to a cockroach db instance. For example, if you are
running a local cockroachdb (with --insecure) and have created
a database called 'linkgraph' you can define the envvar by
running:

export CDB_DSN='postgresql://root@localhost:26257/linkgraph?sslmode=disable'

endef
export dsn_missing_error

check-cdb-env:
ifndef CDB_DSN
	$(error ${dsn_missing_error})
endif