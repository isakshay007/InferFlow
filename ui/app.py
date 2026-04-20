import os
import httpx
import streamlit as st
from datetime import datetime

INFERFLOW_URL = os.getenv(
    "INFERFLOW_URL",
    "http://k8s-default-inferflo-b7faaa4fda-434516550.us-east-1.elb.amazonaws.com",
)
MODEL = os.getenv("INFERFLOW_MODEL", "Qwen/Qwen2.5-0.5B-Instruct")
POLL_INTERVAL = 2

st.set_page_config(
    page_title="InferFlow — LLM Router",
    page_icon="⚡",
    layout="wide",
    initial_sidebar_state="expanded",
)

# ── custom CSS ────────────────────────────────────────────────────────────────
st.markdown("""
<style>
/* fonts + base */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');

html, body, [class*="css"] { font-family: 'Inter', sans-serif; }

/* hide default streamlit header */
#MainMenu, footer, header { visibility: hidden; }

/* tighten top padding so hero sits closer to top */
.block-container {
    padding-top: 1.2rem !important;
    padding-bottom: 0rem !important;
}

/* main bg */
.stApp { background: #0f1117; }

/* hero banner — tighter, sits near top */
.hero {
    background: linear-gradient(135deg, #1a1f2e 0%, #0f1117 50%, #1a1f2e 100%);
    border: 1px solid #2d3748;
    border-radius: 16px;
    padding: 20px 28px;
    margin-bottom: 16px;
}

/* give messages area bottom breathing room so last message isn't hidden behind input */
.chat-messages-container {
    padding-bottom: 20px;
}
.hero h1 {
    font-size: 1.8rem;
    font-weight: 700;
    color: #e2e8f0;
    margin: 0 0 4px 0;
}
.hero p {
    color: #718096;
    font-size: 0.88rem;
    margin: 0;
}
.hero .badge {
    display: inline-block;
    background: #1a2744;
    border: 1px solid #2d4a8a;
    color: #63b3ed;
    border-radius: 20px;
    padding: 3px 12px;
    font-size: 0.78rem;
    font-weight: 500;
    margin: 8px 4px 0 0;
}

/* metric cards */
.metric-card {
    background: #1a1f2e;
    border: 1px solid #2d3748;
    border-radius: 12px;
    padding: 16px 20px;
    text-align: center;
}
.metric-card .label {
    font-size: 0.75rem;
    color: #718096;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 500;
}
.metric-card .value {
    font-size: 1.7rem;
    font-weight: 700;
    color: #e2e8f0;
    margin: 4px 0 0 0;
}
.metric-card .value.green { color: #68d391; }
.metric-card .value.yellow { color: #f6e05e; }
.metric-card .value.red { color: #fc8181; }
.metric-card .value.blue { color: #63b3ed; }

/* backend cards */
.backend-card {
    background: #1a1f2e;
    border: 1px solid #2d3748;
    border-radius: 10px;
    padding: 14px 18px;
    margin-bottom: 8px;
}
.backend-card.healthy { border-left: 3px solid #68d391; }
.backend-card.unhealthy { border-left: 3px solid #fc8181; }
.backend-name {
    font-weight: 600;
    color: #e2e8f0;
    font-size: 0.9rem;
}
.backend-stats {
    color: #718096;
    font-size: 0.8rem;
    margin-top: 2px;
    font-family: 'JetBrains Mono', monospace;
}

/* strategy buttons */
.stButton > button {
    background: #1a1f2e !important;
    border: 1px solid #2d3748 !important;
    color: #a0aec0 !important;
    border-radius: 8px !important;
    font-size: 0.82rem !important;
    font-weight: 500 !important;
    transition: all 0.15s ease !important;
    padding: 6px 12px !important;
}
.stButton > button:hover {
    border-color: #4a5568 !important;
    color: #e2e8f0 !important;
    background: #2d3748 !important;
}

/* active strategy pill */
.active-strategy {
    background: linear-gradient(135deg, #1a3a5c, #1e4080);
    border: 1px solid #2d6bc4;
    color: #63b3ed;
    border-radius: 8px;
    padding: 6px 14px;
    font-size: 0.85rem;
    font-weight: 600;
    display: inline-block;
    font-family: 'JetBrains Mono', monospace;
}

/* routing log entries */
.log-entry {
    background: #1a1f2e;
    border: 1px solid #2d3748;
    border-radius: 8px;
    padding: 10px 16px;
    margin-bottom: 6px;
    font-size: 0.83rem;
    color: #a0aec0;
    font-family: 'JetBrains Mono', monospace;
}
.log-entry .req-num { color: #718096; }
.log-entry .backend-pill {
    background: #1a3a5c;
    color: #63b3ed;
    border-radius: 4px;
    padding: 1px 7px;
    font-size: 0.78rem;
}
.log-entry .strategy-pill {
    background: #1a3320;
    color: #68d391;
    border-radius: 4px;
    padding: 1px 7px;
    font-size: 0.78rem;
}
.log-entry .hit-pill {
    background: #2d2a1a;
    color: #f6e05e;
    border-radius: 4px;
    padding: 1px 7px;
    font-size: 0.78rem;
}

/* chat messages */
.stChatMessage {
    background: #1a1f2e !important;
    border: 1px solid #2d3748 !important;
    border-radius: 12px !important;
}

/* tabs */
.stTabs [data-baseweb="tab-list"] {
    background: #1a1f2e;
    border-radius: 10px;
    padding: 4px;
    gap: 16px;
}
.stTabs [data-baseweb="tab"] {
    background: transparent;
    border-radius: 8px;
    color: #718096;
    font-weight: 500;
    font-size: 0.88rem;
    padding-left: 20px !important;
    padding-right: 20px !important;
}
.stTabs [aria-selected="true"] {
    background: #2d3748 !important;
    color: #e2e8f0 !important;
}

/* sidebar */
section[data-testid="stSidebar"] {
    background: #0d1117 !important;
    border-right: 1px solid #2d3748 !important;
}

/* divider */
hr { border-color: #2d3748 !important; }

/* results table */
.results-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.88rem;
}
.results-table th {
    background: #2d3748;
    color: #a0aec0;
    padding: 10px 16px;
    text-align: left;
    font-weight: 600;
    font-size: 0.78rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
}
.results-table td {
    padding: 10px 16px;
    border-bottom: 1px solid #2d3748;
    color: #e2e8f0;
    font-family: 'JetBrains Mono', monospace;
}
.results-table tr:hover td { background: #1a1f2e; }
.results-table .winner { color: #68d391; font-weight: 700; }
.results-table .loser { color: #fc8181; }

.section-header {
    font-size: 0.75rem;
    font-weight: 600;
    color: #4a5568;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    margin-bottom: 10px;
    margin-top: 4px;
}
</style>
""", unsafe_allow_html=True)

# ── session state ─────────────────────────────────────────────────────────────
for key, default in [
    ("messages", []),
    ("routing_log", []),
    ("request_count", 0),
    ("strategy_toast", None),
]:
    if key not in st.session_state:
        st.session_state[key] = default


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
        r = httpx.put(f"{INFERFLOW_URL}/strategy", json={"strategy": name}, timeout=3)
        return r.status_code == 200
    except Exception:
        return False


SYSTEM_PROMPT = {
    "role": "system",
    "content": (
        "You are a helpful assistant. "
        "Answer the user's question directly. "
        "Do NOT repeat the question, add any prefix, number, or label before your answer. "
        "Start your reply immediately with the answer text."
    ),
}

import re as _re
_JUNK_PREFIX = _re.compile(r'^(assistant\s*:\s*|answer\s*:\s*|response\s*:\s*|[\d\s\?\!\.\:]+)', _re.IGNORECASE)


def chat(messages: list) -> tuple[str, str, str, bool]:
    trimmed = messages[-4:]
    payload = {
        "model": MODEL,
        "messages": [SYSTEM_PROMPT] + trimmed,
        "max_tokens": 300,
        "temperature": 0.7,
        "repetition_penalty": 1.15,
    }
    r = httpx.post(f"{INFERFLOW_URL}/v1/chat/completions", json=payload, timeout=60)
    r.raise_for_status()
    data = r.json()
    reply = data["choices"][0]["message"]["content"].strip()
    # strip leading artifact characters the small model sometimes emits
    reply = _JUNK_PREFIX.sub("", reply).strip() or reply
    backend = r.headers.get("x-inferflow-backend", "unknown")
    strategy = r.headers.get("x-inferflow-strategy", "unknown")
    cache_hit = r.headers.get("x-inferflow-cache-hit", "false").lower() == "true"
    return reply, backend, strategy, cache_hit


STRATEGY_DESCRIPTIONS = {
    "round_robin":   "Cycles backends in order. Simple, no state.",
    "least_pending": "Routes to least-loaded backend. Best tail latency.",
    "random":        "Random selection. Stateless, worst perf.",
    "kv_aware":      "Redis affinity — repeated prompts 22% faster.",
}


# ── sidebar ───────────────────────────────────────────────────────────────────
with st.sidebar:
    st.markdown("""
    <div style="padding: 8px 0 16px 0;">
        <div style="font-size:1.3rem; font-weight:700; color:#e2e8f0;">InferFlow</div>
        <div style="font-size:0.75rem; color:#4a5568; margin-top:2px;">LLM Inference Router</div>
    </div>
    """, unsafe_allow_html=True)

    status_now = fetch_status()
    current_strategy = status_now.get("strategy", "—")
    metrics_now = status_now.get("metrics", {})
    backends_now = status_now.get("backends", [])

    # connection status
    if status_now:
        st.markdown('<div style="display:inline-flex;align-items:center;gap:6px;font-size:0.8rem;color:#68d391;">🟢 <span>Router connected</span></div>', unsafe_allow_html=True)
    else:
        st.markdown('<div style="display:inline-flex;align-items:center;gap:6px;font-size:0.8rem;color:#fc8181;">🔴 <span>Router unreachable</span></div>', unsafe_allow_html=True)

    st.markdown("<div style='height:16px'></div>", unsafe_allow_html=True)
    st.markdown('<div class="section-header">Active Strategy</div>', unsafe_allow_html=True)
    st.markdown(f'<div class="active-strategy">{current_strategy}</div>', unsafe_allow_html=True)

    if current_strategy in STRATEGY_DESCRIPTIONS:
        st.markdown(f'<div style="font-size:0.75rem;color:#4a5568;margin-top:6px;">{STRATEGY_DESCRIPTIONS[current_strategy]}</div>', unsafe_allow_html=True)

    st.markdown("<div style='height:16px'></div>", unsafe_allow_html=True)
    st.markdown('<div class="section-header">Switch Strategy</div>', unsafe_allow_html=True)

    strategies = ["round_robin", "least_pending", "random", "kv_aware"]
    cols = st.columns(2)
    for i, s in enumerate(strategies):
        label = s.replace("_", " ").title()
        if cols[i % 2].button(label, key=f"btn_{s}", use_container_width=True):
            ok = set_strategy(s)
            st.session_state.strategy_toast = f"✓ Switched to **{s}**" if ok else f"✗ Failed to switch"
            st.rerun()

    if st.session_state.strategy_toast:
        st.success(st.session_state.strategy_toast)
        st.session_state.strategy_toast = None

    st.divider()

    # live metrics
    @st.fragment(run_every=POLL_INTERVAL)
    def sidebar_metrics():
        s = fetch_status()
        if not s:
            return
        m = s.get("metrics", {})
        backends = s.get("backends", [])

        st.markdown('<div class="section-header">Live Metrics</div>', unsafe_allow_html=True)

        c1, c2 = st.columns(2)
        c1.metric("Requests", m.get("requests_total", 0))
        c2.metric("In-flight", m.get("in_flight", 0))

        kv_rate = m.get("kv_cache_hit_rate")
        errors = m.get("backend_errors", 0)
        c3, c4 = st.columns(2)
        c3.metric("KV Hit Rate", f"{kv_rate*100:.0f}%" if kv_rate else "—")
        c4.metric("Errors", errors)

        st.markdown("<div style='height:8px'></div>", unsafe_allow_html=True)
        st.markdown('<div class="section-header">Backends</div>', unsafe_allow_html=True)

        for b in backends:
            healthy = b.get("healthy", False)
            status_class = "healthy" if healthy else "unhealthy"
            dot = "🟢" if healthy else "🔴"
            lat = b.get("latency_ms", 0)
            pending = b.get("pending", 0)
            st.markdown(f"""
            <div class="backend-card {status_class}">
                <div class="backend-name">{dot} {b['name']}</div>
                <div class="backend-stats">{pending} pending · {lat} ms avg</div>
            </div>
            """, unsafe_allow_html=True)

        st.markdown(f'<div style="font-size:0.7rem;color:#4a5568;text-align:right;">Updated {datetime.now().strftime("%H:%M:%S")}</div>', unsafe_allow_html=True)

    sidebar_metrics()


# ── hero banner ───────────────────────────────────────────────────────────────
st.markdown("""
<div class="hero">
    <p>Scalable LLM inference router — Go control-plane · AWS EKS · 3× llama.cpp backends · Redis KV-cache affinity</p>
    <span class="badge">Qwen2.5-0.5B-Instruct</span>
    <span class="badge">4 routing strategies</span>
    <span class="badge">OpenAI-compatible API</span>
    <span class="badge">Live on EKS</span>
</div>
""", unsafe_allow_html=True)


# ── tabs ──────────────────────────────────────────────────────────────────────
tab1, tab2, tab3 = st.tabs(["💬   Chat", "📊   Benchmark Results", "🔀   Routing Log"])


# ══ TAB 1: CHAT ═══════════════════════════════════════════════════════════════
with tab1:
    # scrollable messages area — fixed height pushes chat_input to the bottom
    msgs_box = st.container(height=380, border=False)
    with msgs_box:
        if not st.session_state.messages:
            st.markdown("""
            <div style="text-align:center;padding:100px 20px;color:#4a5568;">
                <div style="font-size:0.88rem;color:#4a5568;">Send a message below to test the router.</div>
            </div>
            """, unsafe_allow_html=True)
        for msg in st.session_state.messages:
            label = "You" if msg["role"] == "user" else "Agent"
            avatar = "🧑" if msg["role"] == "user" else "🤖"
            with st.chat_message(label, avatar=avatar):
                st.markdown(msg["content"])
                if msg["role"] == "assistant" and "backend" in msg:
                    cache_badge = ""
                    if msg.get("strategy") == "kv_aware":
                        hit = msg.get("cache_hit", False)
                        cache_badge = ' · <span style="color:#f6e05e;">⚡ KV cache hit</span>' if hit else ' · <span style="color:#718096;">cache miss</span>'
                    st.markdown(
                        f'<div style="font-size:0.75rem;color:#4a5568;margin-top:4px;font-family:\'JetBrains Mono\',monospace;">'
                        f'→ <span style="color:#63b3ed;">{msg["backend"]}</span> '
                        f'via <span style="color:#68d391;">{msg["strategy"]}</span>'
                        f'{cache_badge}</div>',
                        unsafe_allow_html=True,
                    )

    if st.session_state.messages:
        _, col_btn = st.columns([9, 1])
        with col_btn:
            if st.button("Clear", key="clear_chat", use_container_width=True):
                st.session_state.messages = []
                st.rerun()

    if prompt := st.chat_input("Ask anything — watch the router decide which backend responds…"):
        st.session_state.messages.append({"role": "user", "content": prompt})

        # show spinner inside the messages box while waiting for the backend
        with msgs_box:
            with st.chat_message("Agent", avatar="🤖"):
                with st.spinner("Routing request…"):
                    try:
                        reply, backend, strategy, cache_hit = chat(st.session_state.messages)
                    except Exception as e:
                        reply = f"Error: {e}"
                        backend, strategy, cache_hit = "—", "—", False

        st.session_state.messages.append({
            "role": "assistant",
            "content": reply,
            "backend": backend,
            "strategy": strategy,
            "cache_hit": cache_hit,
        })

        # update routing log
        st.session_state.request_count += 1
        latest = fetch_status()
        kv_rate = latest.get("metrics", {}).get("kv_cache_hit_rate")
        kv_str = f"{kv_rate*100:.0f}%" if kv_rate is not None else "—"
        st.session_state.routing_log.insert(0, {
            "n": st.session_state.request_count,
            "backend": backend,
            "strategy": strategy,
            "cache_hit": cache_hit,
            "kv_rate": kv_str,
            "time": datetime.now().strftime("%H:%M:%S"),
        })
        st.session_state.routing_log = st.session_state.routing_log[:15]

        # rerun so all messages re-render cleanly inside msgs_box
        st.rerun()


# ══ TAB 2: BENCHMARK RESULTS ══════════════════════════════════════════════════
with tab2:
    st.markdown('<div style="height:8px"></div>', unsafe_allow_html=True)

    st.markdown("""
    <div style="background:#1a1f2e;border:1px solid #2d3748;border-radius:12px;padding:20px 24px;margin-bottom:20px;">
        <div style="font-size:0.8rem;color:#718096;">
            Load tests run on live AWS EKS cluster — 3× c5.xlarge nodes, llama.cpp + Qwen2.5-0.5B-Instruct.
            400 total requests (100 per strategy), concurrency 3, mixed prompts.
        </div>
    </div>
    """, unsafe_allow_html=True)

    # strategy comparison table
    st.markdown('<div class="section-header">Experiment 1 — Strategy Latency Comparison</div>', unsafe_allow_html=True)
    st.markdown("""
    <table class="results-table">
        <thead>
            <tr>
                <th>Strategy</th>
                <th>p50 (ms)</th>
                <th>p95 (ms)</th>
                <th>Max (ms)</th>
                <th>Backend Spread</th>
                <th>Verdict</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td style="font-family:'JetBrains Mono',monospace;">round_robin</td>
                <td>4,693</td>
                <td>5,296</td>
                <td>7,995</td>
                <td>34 / 33 / 33</td>
                <td style="color:#a0aec0;">Baseline</td>
            </tr>
            <tr style="background:#0f1a0f;">
                <td style="font-family:'JetBrains Mono',monospace;color:#68d391;font-weight:700;">least_pending</td>
                <td class="winner">4,757</td>
                <td class="winner">4,860</td>
                <td class="winner">4,995</td>
                <td class="winner">33 / 33 / 34</td>
                <td><span style="background:#1a3320;color:#68d391;border-radius:4px;padding:2px 8px;font-size:0.75rem;font-weight:600;">✓ WINNER</span></td>
            </tr>
            <tr>
                <td style="font-family:'JetBrains Mono',monospace;">kv_aware</td>
                <td>4,812</td>
                <td style="color:#f6e05e;">6,906</td>
                <td style="color:#f6e05e;">8,930</td>
                <td style="color:#f6e05e;">46 / 30 / 24</td>
                <td style="color:#f6e05e;">Hotspot</td>
            </tr>
            <tr>
                <td style="font-family:'JetBrains Mono',monospace;">random</td>
                <td class="loser">5,949</td>
                <td class="loser">7,789</td>
                <td class="loser">9,046</td>
                <td>35 / 33 / 32</td>
                <td><span style="background:#2d1a1a;color:#fc8181;border-radius:4px;padding:2px 8px;font-size:0.75rem;font-weight:600;">✗ WORST</span></td>
            </tr>
        </tbody>
    </table>
    """, unsafe_allow_html=True)

    st.markdown("<div style='height:24px'></div>", unsafe_allow_html=True)

    # kv cache table
    st.markdown('<div class="section-header">Experiment 2 — KV Cache Affinity Benefit (concurrency 1, repeated 200-token prompt)</div>', unsafe_allow_html=True)
    st.markdown("""
    <table class="results-table">
        <thead>
            <tr><th>Strategy</th><th>Request Type</th><th>Avg (ms)</th><th>Min (ms)</th><th>Max (ms)</th><th>n</th></tr>
        </thead>
        <tbody>
            <tr>
                <td style="font-family:'JetBrains Mono',monospace;">round_robin</td>
                <td>Repeated prompt</td>
                <td class="loser">6,042</td>
                <td>4,785</td>
                <td class="loser">6,962</td>
                <td>5</td>
            </tr>
            <tr>
                <td style="font-family:'JetBrains Mono',monospace;">round_robin</td>
                <td>Unique prompt</td>
                <td>4,657</td>
                <td>4,578</td>
                <td>4,758</td>
                <td>5</td>
            </tr>
            <tr style="background:#0f1a0f;">
                <td style="font-family:'JetBrains Mono',monospace;color:#68d391;font-weight:700;">kv_aware</td>
                <td>Repeated prompt</td>
                <td class="winner">4,708</td>
                <td class="winner">4,537</td>
                <td class="winner">4,813</td>
                <td>10</td>
            </tr>
        </tbody>
    </table>
    """, unsafe_allow_html=True)

    st.markdown("""
    <div style="margin-top:16px;background:#1a2a1a;border:1px solid #2d4a2d;border-radius:10px;padding:16px 20px;">
        <span style="color:#68d391;font-weight:700;font-size:0.9rem;">Key Finding: 22% faster</span>
        <div style="color:#718096;font-size:0.82rem;margin-top:6px;">
        kv_aware routes repeated prompts to the same backend via Redis affinity. That backend has the KV attention cache warm in RAM —
        completing at 4,708ms vs 6,042ms for round_robin on the same prompt. Matches fresh unique-prompt speed.
        </div>
    </div>
    """, unsafe_allow_html=True)

    st.markdown("<div style='height:24px'></div>", unsafe_allow_html=True)

    # charts
    st.markdown('<div class="section-header">Charts</div>', unsafe_allow_html=True)
    c1, c2 = st.columns(2)

    import os as _os
    chart_dir = _os.path.join(_os.path.dirname(__file__), "..", "results")

    for path, caption, col in [
        ("latency_bars.png",        "p50 / p95 Latency by Strategy",        c1),
        ("latency_cdf.png",         "Latency CDF — random's long tail",      c2),
        ("backend_distribution.png","Backend Distribution per Strategy",     c1),
        ("latency_over_time.png",   "Latency over Time",                     c2),
        ("kv_hit_rate.png",         "KV Cache Hit Rate (kv_aware = 100%)",   c1),
    ]:
        full = _os.path.join(chart_dir, path)
        if _os.path.exists(full):
            col.image(full, caption=caption, use_container_width=True)


# ══ TAB 3: ROUTING LOG ════════════════════════════════════════════════════════
with tab3:
    st.markdown("<div style='height:8px'></div>", unsafe_allow_html=True)

    if not st.session_state.routing_log:
        st.markdown("""
        <div style="text-align:center;padding:60px 20px;color:#4a5568;">
            <div style="font-size:2rem;margin-bottom:12px;">🔀</div>
            <div style="font-size:0.9rem;">No requests yet — send a message in the Chat tab to see routing decisions here.</div>
        </div>
        """, unsafe_allow_html=True)
    else:
        st.markdown(f'<div style="font-size:0.8rem;color:#4a5568;margin-bottom:12px;">{len(st.session_state.routing_log)} routing decisions recorded</div>', unsafe_allow_html=True)

        for entry in st.session_state.routing_log:
            cache_badge = ""
            if entry.get("strategy") == "kv_aware":
                if entry.get("cache_hit"):
                    cache_badge = '<span class="hit-pill">⚡ cache hit</span>'
                else:
                    cache_badge = '<span style="color:#4a5568;font-size:0.75rem;">cache miss</span>'

            st.markdown(f"""
            <div class="log-entry">
                <span class="req-num">#{entry['n']:03d}</span>
                &nbsp;→&nbsp;
                <span class="backend-pill">{entry['backend']}</span>
                &nbsp;via&nbsp;
                <span class="strategy-pill">{entry['strategy']}</span>
                &nbsp;&nbsp;{cache_badge}
                <span style="float:right;color:#4a5568;font-size:0.72rem;">{entry['time']} · KV {entry['kv_rate']}</span>
            </div>
            """, unsafe_allow_html=True)

        if st.button("Clear log", key="clear_log"):
            st.session_state.routing_log = []
            st.rerun()
