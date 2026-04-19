import os
import httpx
import streamlit as st
from datetime import datetime

INFERFLOW_URL = os.getenv(
    "INFERFLOW_URL",
    "http://localhost:8080",
)
MODEL = os.getenv("INFERFLOW_MODEL", "Qwen/Qwen2.5-0.5B-Instruct")
POLL_INTERVAL = 2  # seconds

st.set_page_config(
    page_title="InferFlow",
    page_icon="⚡",
    layout="wide",
)

# ── session state defaults ────────────────────────────────────────────────────
if "messages" not in st.session_state:
    st.session_state.messages = []
if "routing_log" not in st.session_state:
    st.session_state.routing_log = []
if "request_count" not in st.session_state:
    st.session_state.request_count = 0
if "strategy_toast" not in st.session_state:
    st.session_state.strategy_toast = None


# ── helpers ───────────────────────────────────────────────────────────────────

def fetch_status() -> dict:
    try:
        r = httpx.get(f"{INFERFLOW_URL}/api/status", timeout=2)
        if r.status_code == 200:
            return r.json()
    except Exception:
        pass
    return {}


def set_strategy(name: str) -> bool:
    try:
        r = httpx.put(
            f"{INFERFLOW_URL}/strategy",
            json={"strategy": name},
            timeout=3,
        )
        return r.status_code == 200
    except Exception:
        return False


SYSTEM_PROMPT = {
    "role": "system",
    "content": "You are a helpful assistant. Answer concisely and accurately.",
}


def chat(messages: list) -> tuple[str, str, str]:
    """POST /v1/chat/completions (non-streaming). Returns (reply, backend, strategy)."""
    # cap context to last 6 messages to avoid confusing the small model
    trimmed = messages[-6:]
    payload = {
        "model": MODEL,
        "messages": [SYSTEM_PROMPT] + trimmed,
        "max_tokens": 300,
        "temperature": 0.7,
        "repetition_penalty": 1.15,
    }
    r = httpx.post(
        f"{INFERFLOW_URL}/v1/chat/completions",
        json=payload,
        timeout=60,
    )
    r.raise_for_status()
    data = r.json()
    reply = data["choices"][0]["message"]["content"]
    backend = r.headers.get("x-inferflow-backend", "unknown")
    strategy = r.headers.get("x-inferflow-strategy", "unknown")
    return reply, backend, strategy


# ── sidebar ───────────────────────────────────────────────────────────────────

with st.sidebar:
    st.title("⚡ InferFlow")
    st.caption(f"Router: `{INFERFLOW_URL}`")
    st.divider()

    # ── Strategy switcher (INF-33) ────────────────────────────────────────────
    st.subheader("Routing Strategy")
    strategies = ["round_robin", "least_pending", "random", "kv_aware"]

    status_now = fetch_status()
    current_strategy = status_now.get("strategy", "—")
    st.caption(f"Active: **{current_strategy}**")

    cols = st.columns(2)
    for i, s in enumerate(strategies):
        label = s.replace("_", " ").title()
        if cols[i % 2].button(label, key=f"btn_{s}", use_container_width=True):
            ok = set_strategy(s)
            if ok:
                st.session_state.strategy_toast = f"Switched to **{s}**"
            else:
                st.session_state.strategy_toast = f"Failed to switch to {s}"
            st.rerun()

    if st.session_state.strategy_toast:
        st.success(st.session_state.strategy_toast)
        st.session_state.strategy_toast = None

    st.divider()

    # ── Live metrics panel (INF-34) ───────────────────────────────────────────
    st.subheader("Live Metrics")

    @st.fragment(run_every=POLL_INTERVAL)
    def metrics_panel():
        s = fetch_status()
        if not s:
            st.warning("Router unreachable")
            return

        metrics = s.get("metrics", {})
        backends = s.get("backends", [])

        col1, col2 = st.columns(2)
        col1.metric("Strategy", s.get("strategy", "—"))
        col2.metric("In-flight", metrics.get("in_flight", 0))

        c1, c2 = st.columns(2)
        c1.metric("Total Requests", metrics.get("requests_total", 0))
        c2.metric("Backend Errors", metrics.get("backend_errors", 0))

        kv_rate = metrics.get("kv_cache_hit_rate")
        if kv_rate is not None:
            st.metric("KV Cache Hit Rate", f"{kv_rate * 100:.1f}%")

        st.caption("**Workers**")
        for b in backends:
            health_icon = "🟢" if b.get("healthy") else "🔴"
            st.markdown(
                f"{health_icon} **{b['name']}** — "
                f"{b.get('pending', b.get('selections', 0))} pending · "
                f"{b.get('latency_ms', '?')} ms"
            )

        st.caption(f"Updated {datetime.now().strftime('%H:%M:%S')}")

    metrics_panel()


# ── main: chat (INF-33) ───────────────────────────────────────────────────────

st.header("Chat")

for msg in st.session_state.messages:
    with st.chat_message(msg["role"]):
        st.markdown(msg["content"])
        if msg["role"] == "assistant" and "backend" in msg:
            st.caption(
                f"Backend: `{msg['backend']}` via `{msg['strategy']}`"
            )

if prompt := st.chat_input("Send a message…"):
    st.session_state.messages.append({"role": "user", "content": prompt})
    with st.chat_message("user"):
        st.markdown(prompt)

    with st.chat_message("assistant"):
        with st.spinner("Thinking…"):
            try:
                reply, backend, strategy = chat(st.session_state.messages)
            except Exception as e:
                reply = f"Error: {e}"
                backend = "—"
                strategy = "—"
        st.markdown(reply)
        st.caption(f"Backend: `{backend}` via `{strategy}`")

    st.session_state.messages.append(
        {"role": "assistant", "content": reply, "backend": backend, "strategy": strategy}
    )

    # log routing decision
    st.session_state.request_count += 1
    latest = fetch_status()
    kv_rate = latest.get("metrics", {}).get("kv_cache_hit_rate")
    kv_hit = f"{kv_rate * 100:.0f}%" if kv_rate is not None else "—"
    st.session_state.routing_log.insert(
        0,
        f"Request #{st.session_state.request_count} → **{backend}** via `{strategy}` (KV hit rate {kv_hit})",
    )
    st.session_state.routing_log = st.session_state.routing_log[:10]

# ── routing decision log (INF-35) ────────────────────────────────────────────

if st.session_state.routing_log:
    st.divider()
    st.subheader("Routing Decision Log")
    for entry in st.session_state.routing_log:
        st.markdown(f"- {entry}")
