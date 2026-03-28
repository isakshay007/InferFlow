# Local Development

InferFlow supports a fast local development loop using the mock backend.

## Native Run

Start the mock backend:

```bash
go run ./cmd/mock-backend
```

Start the router in another terminal:

```bash
$env:INFERFLOW_BACKENDS="http://localhost:9000"
go run ./cmd/router
```

Send a sample request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"mock-llm\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello from InferFlow\"}]}"
```

## Docker Compose

```bash
docker compose up --build
```

The router listens on `http://localhost:8080`.

## Tests

Run Go tests:

```bash
go test ./...
```

Generate sample load:

```bash
python loadgen/generator.py --requests 5 --output results/sample.csv
```

## Notes

- The local workflow is mock-backed by default.
- The first real Triton runtime target is AWS GPU, not local Triton.
