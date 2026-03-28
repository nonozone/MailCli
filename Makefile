.PHONY: build test demo-local-thread-refresh demo-local-thread-check

MAILCLI_BIN ?= /tmp/mailcli
FIXTURES_CONFIG ?= examples/config/fixtures-dir.yaml
FIXTURES_ACCOUNT ?= fixtures
FIXTURES_INDEX ?= /tmp/mailcli-fixtures-index.json
LOCAL_THREAD_DEMO_DIR ?= examples/artifacts/local-thread-demo

build:
	go build -o $(MAILCLI_BIN) ./cmd/mailcli

test:
	go test ./...

demo-local-thread-refresh: build
	PYTHONDONTWRITEBYTECODE=1 python3 examples/python/refresh_local_thread_demo.py \
		--mailcli-bin $(MAILCLI_BIN) \
		--config $(FIXTURES_CONFIG) \
		--account $(FIXTURES_ACCOUNT) \
		--index $(FIXTURES_INDEX) \
		--output-dir $(LOCAL_THREAD_DEMO_DIR)

demo-local-thread-check: build
	PYTHONDONTWRITEBYTECODE=1 python3 examples/python/refresh_local_thread_demo.py \
		--mailcli-bin $(MAILCLI_BIN) \
		--config $(FIXTURES_CONFIG) \
		--account $(FIXTURES_ACCOUNT) \
		--index $(FIXTURES_INDEX) \
		--output-dir $(LOCAL_THREAD_DEMO_DIR) \
		--check
