TEST_COVER=test.cover

# for in-docker build
BUILD_ROOT=$(shell pwd)
BUILD_BASE=`realpath --relative-to=$(GOPATH)/src $(BUILD_ROOT)`

PROJECT=`basename $(BUILD_ROOT)`
PROJECT_MASK=$(shell echo -n "'$(PROJECT)*'")

# they both will be written into binaries
GIT_HASH=`git rev-parse --short HEAD`
BUILD_DATE=`date +%FT%T%z`

# linker configs (default/release)
LDFLAGS=-X "main.GitHash=$(GIT_HASH)" -X "main.BuildAt=$(BUILD_DATE)"
LDFLAGS_REL=-w -s $(LDFLAGS)

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

APPs = toggle-svc

build: vet
	@- $(foreach APP, $(APPs), \
		echo -n "[build] $(PROJECT): $(APP) .." ; \
		go build -ldflags "$(LDFLAGS)" -o "bin/$(APP)" "./cmd/$(APP)" && echo "\b\bok" ; \
	)

release: vet
	@- $(foreach APP, $(APPs), \
		echo -n "[release] $(PROJECT): $(APP) .." ; \
		go build -ldflags "$(LDFLAGS_REL)" -o "bin/$(APP)" "./cmd/$(APP)" && echo "\b\bok" ; \
	)

mod:
	@- echo "[mod] verify .."
	@- go mod verify

vet:
	@- echo -n "[tool] vet .."
	@- go vet ./... && echo "\b\bok"

lint: vet
	@- golangci-lint run

test: vet
	@- CGO_ENABLED=1 go test -race -count 1 -v -coverprofile="$(TEST_COVER)" ./...

test-cover: test
	@- go tool cover -func="$(TEST_COVER)"

docker-app:
	@- echo "[app-pack] for $(BUILD_BASE)"
	@- docker build \
		--build-arg "BUILD_BASE=$(BUILD_BASE)" \
		-t "$(PROJECT):$(GIT_HASH)" \
		-t "$(PROJECT):latest" \
		-f ./docker/Dockerfile.app .

docker-build: docker-vet docker-app
	@- docker-compose \
		build --parallel --force-rm

docker-clean: docker-vet
	@- docker-compose \
		-f docker-compose.yml \
		down -v --remove-orphans --rmi local
	@- docker rmi -f \
		`docker images -q --filter=reference=$(PROJECT_MASK)`

docker-vet:
	@- echo -n "[tool] docker-compose vet .."
	@- docker-compose \
		-f docker-compose.yml \
		config -q && echo "\b\bok"

docker-dev-up: docker-vet
	@- docker-compose \
		-f docker-compose.yml \
		up -d

docker-dev-down:
	@- docker-compose \
		-f docker-compose.yml \
		down

clean:
	@- echo "[clean] covers"
	@- rm -f "$(TEST_COVER)"
	@- echo "[clean] apps"
	@- $(foreach APP, $(APPs), \
		rm -f "bin/$(APP)" ; \
	)
