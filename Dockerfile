FROM golang:1.22 as tester
WORKDIR app

# Install deps
COPY go.mod go.sum Makefile ./
COPY configs/.docker-test.env configs/.env
RUN go mod download
RUN make migrate-check-deps

# Run
COPY . .
RUN mv configs/.docker-test.env configs/.env
ENTRYPOINT make migration-up && go test ./...


