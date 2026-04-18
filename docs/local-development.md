# Local Development

InferFlow supports a fast local development loop using the mock backend while keeping the router strategy surface identical to the EKS/vLLM deployment path.

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

Run all active strategies:

```bash
python loadgen/generator.py --requests 5 --strategies round_robin,least_pending,random,kv_aware --output results/strategies.csv
```

## Notes

- The local workflow is mock-backed by default.
- The production-like cloud path is EKS + vLLM.
- If `INFERFLOW_REDIS_ADDR` is not set, `kv_aware` uses the in-memory affinity store locally.
