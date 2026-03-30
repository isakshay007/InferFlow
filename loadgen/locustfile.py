import os

from locust import HttpUser, constant, task


class InferFlowUser(HttpUser):
    wait_time = constant(0)

    @task
    def chat_completion(self) -> None:
        payload = {
            "model": os.getenv("INFERFLOW_MODEL", "local-llm"),
            "messages": [{"role": "user", "content": "Name 2 colors."}],
            "stream": False,
        }

        with self.client.post(
            "/v1/chat/completions",
            json=payload,
            name="POST /v1/chat/completions",
            catch_response=True,
            timeout=70,
        ) as resp:
            if resp.status_code != 200:
                resp.failure(f"status={resp.status_code}")
                return

            try:
                body = resp.json()
                choices = body.get("choices", [])
                content = choices[0]["message"]["content"] if choices else ""
                if not str(content).strip():
                    resp.failure("empty assistant response")
                    return
            except Exception as exc:  # noqa: BLE001
                resp.failure(f"invalid json: {exc}")
                return

            resp.success()
