SHELL := /bin/bash
ROOT := $(CURDIR)
GO := $(if $(wildcard .tools/go/bin/go),$(ROOT)/.tools/go/bin/go,go)
FLUTTER := $(if $(wildcard .tools/flutter/bin/flutter),$(ROOT)/.tools/flutter/bin/flutter,flutter)
DART := $(if $(wildcard .tools/flutter/bin/dart),$(ROOT)/.tools/flutter/bin/dart,dart)
export PATH := $(ROOT)/.tools/go/bin:$(ROOT)/.tools/flutter/bin:$(PATH)
export JAVA_HOME := $(if $(wildcard .tools/java),$(ROOT)/.tools/java,$(JAVA_HOME))
export ANDROID_HOME := $(if $(wildcard .tools/android),$(ROOT)/.tools/android,$(ANDROID_HOME))
.PHONY: setup db-up migrate migrate-down seed-admin backend public public-lan admin-api worker admin mobile contracts test test-api build android-share android-debug android-bundle
setup:
	node scripts/setup.mjs
	cd admin-web && npm ci
	cd mobile-app && $(FLUTTER) pub get
	cd mobile-app/tooling/storage_tests && $(FLUTTER) pub get
db-up:
	@docker inspect doodhee-dev-db >/dev/null 2>&1 && docker start doodhee-dev-db || docker compose up -d postgres
migrate:
	set -a; source .env; set +a; cd backend && $(GO) run ./cmd/migrate
migrate-down:
	@test "$(CONFIRM)" = "disposable-database" || (echo 'Set CONFIRM=disposable-database; this deletes all Doodhee tables.'; exit 1)
	set -a; source .env; set +a; export CONFIRM=disposable-database; cd backend && $(GO) run ./cmd/migrate down
seed-admin:
	set -a; source .env; set +a; cd backend && $(GO) run ./cmd/seed-admin
public:
	set -a; source .env; set +a; cd backend && $(GO) run ./cmd/public-gateway
public-lan:
	@test -n "$$LAN_IP" || (echo "Set LAN_IP to this computer's Wi-Fi IPv4 address, for example: LAN_IP=192.168.1.10 make public-lan"; exit 1)
	set -a; source .env; set +a; export PUBLIC_HTTP_ADDR="$$LAN_IP:18080"; cd backend && $(GO) run ./cmd/public-gateway
admin-api:
	set -a; source .env; set +a; cd backend && $(GO) run ./cmd/admin-gateway
worker:
	set -a; source .env; set +a; cd backend && $(GO) run ./cmd/event-worker
backend:
	$(MAKE) -j3 public admin-api worker
admin:
	cd admin-web && npm run dev
mobile:
	cd mobile-app && $(FLUTTER) run --dart-define=PUBLIC_API_URL=$${PUBLIC_API_URL:-http://10.0.2.2:18080}
contracts:
	cd backend && $(GO) run ./cmd/contracts
	node scripts/generate-clients.mjs
	cd mobile-app && $(DART) format lib/generated
test:
	set -a; source .env; set +a; export TEST_DATABASE_URL=$$DATABASE_URL; cd backend && $(GO) test ./... && $(GO) vet ./...
	cd admin-web && npm run typecheck && npm test && npm run lint
	cd mobile-app && $(FLUTTER) analyze && $(FLUTTER) test
	cd mobile-app/tooling/storage_tests && $(FLUTTER) pub get && $(FLUTTER) analyze && $(FLUTTER) test
test-api:
	set -a; source .env; set +a; export TEST_DATABASE_URL=$${TEST_DATABASE_URL:-$$DATABASE_URL}; export API_REPORT_DIR=$${API_REPORT_DIR:-$(ROOT)/artifacts/api-verification/final}; cd backend && $(GO) test -race ./internal/adapters/httpapi -run '^TestAllAPI' -count=1 -v
build:
	mkdir -p artifacts
	cd backend && $(GO) build -o ../artifacts/public-gateway ./cmd/public-gateway && $(GO) build -o ../artifacts/admin-gateway ./cmd/admin-gateway && $(GO) build -o ../artifacts/event-worker ./cmd/event-worker
	cd admin-web && npm run build
	$(MAKE) android-share
android-share:
	@test -n "$$PUBLIC_API_URL" || (echo 'Set PUBLIC_API_URL to a backend reachable from the phone. See docs/PHONE_TESTING.md.'; exit 1)
	mkdir -p artifacts/android/symbols
	cd mobile-app && $(FLUTTER) build apk --release --split-per-abi --target-platform android-arm,android-arm64 --split-debug-info=$(ROOT)/artifacts/android/symbols -PlocalTestSigning=true -PallowLocalHttp=true -PcompactNativeLibraries=true --dart-define=PUBLIC_API_URL=$$PUBLIC_API_URL --dart-define="BUILD_LABEL=$${BUILD_LABEL:-Testing build}"
	cp mobile-app/build/app/outputs/flutter-apk/app-arm64-v8a-release.apk artifacts/android/doodhee-arm64-testing.apk
	cp mobile-app/build/app/outputs/flutter-apk/app-armeabi-v7a-release.apk artifacts/android/doodhee-arm32-testing.apk
	cp mobile-app/build/app/outputs/mapping/release/mapping.txt artifacts/android/symbols/android-r8-mapping.txt
	node scripts/apk-size-report.mjs artifacts/android/doodhee-arm64-testing.apk artifacts/android/doodhee-arm32-testing.apk > artifacts/android/size-report.json
android-debug:
	cd mobile-app && $(FLUTTER) build apk --debug --split-per-abi --target-platform android-arm64 --dart-define=PUBLIC_API_URL=$${PUBLIC_API_URL:-http://10.0.2.2:18080}
android-bundle:
	cd mobile-app && $(FLUTTER) build appbundle --debug --dart-define=PUBLIC_API_URL=$${PUBLIC_API_URL:-http://10.0.2.2:18080}
