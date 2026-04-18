GO ?= go
PYTHON ?= python

.PHONY: test router mock-backend vllm-adapter loadgen-sample loadgen-strategies compose-up compose-down terraform-validate

test:
	$(GO) test ./...

router:
	$(GO) run ./cmd/router

mock-backend:
	$(GO) run ./cmd/mock-backend

vllm-adapter:
	$(GO) run ./cmd/vllm-adapter

loadgen-sample:
	$(PYTHON) loadgen/generator.py --requests 5 --output results/sample.csv

loadgen-strategies:
	$(PYTHON) loadgen/generator.py --requests 5 --strategies round_robin,least_pending,random,kv_aware --output results/strategies.csv

compose-up:
	docker compose up --build

compose-down:
	docker compose down --remove-orphans

terraform-validate:
	cd terraform/environments/aws && terraform init -backend=false && terraform validate
