# InferFlow — Final Submission Verification Checklist
**Team:** Ajin Frank Justin · Akshay Keerthi Adhikasavan Suresh
**Date:** 2026-04-20

Each item below is graded against what the assignment requires, followed by a verdict and any gaps.

---

## [5 marks] Item 1 — Video

**Asset:** SharePoint recording — https://northeastern-my.sharepoint.com/:v:/g/personal/justin_aj_northeastern_edu/IQA8101Jx72VTZM8HJT8JErIAWOg9XlDhyvdseugW2Yl9Ww?e=yVx9IZ

**Requirement:** 2–3 min per team member, on the part they worked on, real voice (no AI voiceover), screen capture.

**Verdict:** ✅ Asset exists. **I can't verify video content from here.** Confirm manually:
- Both Ajin and Akshay appear / speak live (real voice)
- Each segment is 2–3 minutes
- Each covers the contributor's actual work (Ajin: router, kv_aware, Terraform/EKS, load tests; Akshay: UI, metrics, backend integration)
- Link permissions allow the graders to open it (Northeastern tenant — test in incognito)

---

## [5 marks] Item 2 — Code

**Asset:** https://github.com/isakshay007/InferFlow

**Requirement:** Shared repo (individual repos also OK), README introduces the project and the "why," and shows activity over time.

**Verdict:** ✅ Strong.
- README (215 lines) leads with a project intro, quick-start, API surface, and embedded experiment results with charts — introduces the project and explains what was built.
- **18 commits** on main across **2 authors** (Ajin Frank Justin, akshay AS) — co-authorship is visible.
- **6 feature branches** shipped (`triton-setup`, `ui-backend`, `ui-streamlit`, `week2-router-strategies`, `eks-cluster-baseline`, `demo`) — shows progression over time.
- **2 merged PRs** (#1, #2 from `feature/ui-backend`) — shows review workflow.
- Commit trail reflects every major phase: Triton → vLLM → llama.cpp pivot, ALB/IAM setup, UI work, load test results.

**Gaps / consider before submit:**
- Current working branch is `feature/ui-streamlit`. Merge it to `main` (or at least push it) so graders see the final state on the default branch.
- README's "why we built it" pitch is thin — one paragraph would help. The current intro is technical-first. (Optional.)
- GitHub Issues: repo activity is commit/PR-heavy but we don't have GitHub issues. The Linear export (item 3) compensates.

---

## [5 marks] Item 3 — Project Management

**Asset:** Linear export — `Export Sun Apr 19 2026.csv` (Linear issues INF-5 through INF-40, 36 tracked issues + header row).

**Requirement:** Show the path from initial design to final state — problem breakdown, who did what, problems encountered.

**Verdict:** ✅ Excellent. The CSV covers all six phases cleanly:
1. **Phase 1 — System Design & Architecture** (INF-5, INF-6): Go + Redis + vLLM stack decision, OpenAI-compatible router architecture.
2. **Phase 2 — Core Router** (INF-7 to INF-9): HTTP router, mock backend, health/readiness/metrics endpoints.
3. **Phase 3 — Routing Strategies** (INF-10 to INF-15): All four strategies + runtime switching + integration tests.
4. **Phase 4 — Cloud Infrastructure & Pivots** (INF-16 to INF-22): **Shows the DigitalOcean → GCP/GKE → AWS EKS pivot explicitly**, with GPU-quota block as a canceled ticket. This is exactly the "problems encountered along the way" the rubric asks for.
5. **Phase 5 — EKS Deployment** (INF-23 to INF-28): SSO refresh, Terraform apply, Docker builds, ALB, cluster health.
6. **Phase 6 — UI & Load Testing** (INF-29 to INF-40): Metrics endpoints, Streamlit UI, X-Inferflow-Cache-Hit header fix, load generator fix, report.

Assignees alternate between `ajinfrankj@gmail.com` and `adhikasavansuresh.a@northeastern.edu` — who-did-what is clear.

**Gaps:** None. This is the strongest single artifact.

---

## [10 marks] Item 4 — Experiments Report

**Asset:** `InferFlow Experiments Report (1).pdf` — 4 pages.

**Requirement:** ≤ 5 pages; for each experiment: purpose/tradeoff/limitations, detailed results with charts, analysis of evidence.

**Verdict:** ✅ Meets every rubric point.

| Rubric requirement | Experiment 1 (Strategy Latency) | Experiment 2 (KV Cache Benefit) |
|---|---|---|
| Purpose | ✅ | ✅ |
| Tradeoff explored | ✅ routing overhead vs. load balance | ✅ cache reuse vs. load balance |
| Limitations | ✅ 3 sections listed | ✅ 3 sections listed |
| Results with charts | ✅ table + 4 figures | ✅ table + 1 figure + key-finding callout |
| Analysis / conclusions | ✅ per-strategy | ✅ with scaling hypothesis |

Also includes a consolidated "Conclusions" section and a "Reproducibility" section with exact CLI to replay the experiments. Under 5 pages.

**Nit (optional, don't block submission):** The report says Redis cache hits were 100% but doesn't explicitly state the n=10 sample size inside the conclusions. The limitation is acknowledged in the prose — fine as-is.

---

## [5 marks] Item 5 — Community Contributions (Piazza)

**Asset:** [piazza_community_contributions.md](piazza_community_contributions.md) — ready to paste into Piazza.

**Requirement:** Post video + experiments report on Piazza under Final Projects; link to 3 other similar final projects noting similarities, differences, and what you learned.

**Verdict:** ✅ Draft is complete. Three projects linked with both Piazza and repo URLs:
- Yalin Sun — RAG Q&A — https://piazza.com/class/mk3hftotl6e229/post/1443
- Owl — Video Inference — https://piazza.com/class/mk3hftotl6e229/post/1444
- ChoreMate — https://piazza.com/class/mk3hftotl6e229/post/1434

Each comparison covers similarities, differences, and a concrete takeaway. Ends with a cross-cutting observation ("caching helps the happy path, hurts under concurrency") that ties all four projects together.

**Before posting on Piazza, replace these placeholders in the draft:**
- `<sharepoint link>` → actual SharePoint URL
- `<attached PDF>` → attach the report file to the Piazza post
- `<attached Excel export>` → attach the CSV

---

## [5 marks] Item 6 — Lessons Learned (separate submissions, per teammate)

**Assets:**
- [lessons_learned_ajin.md](lessons_learned_ajin.md) — Ajin's reflection
- [lessons_learned_akshay.md](lessons_learned_akshay.md) — Akshay's reflection

**Requirement:** Each team member reflects on what was learned — what went wrong, why, how you'd handle it next time, connecting back to course concepts.

**Verdict:** ✅ Both drafts grounded in actual incidents from the project:
- **Ajin:** cache-hit arithmetic bug (`is_repeat` metric); llama.cpp single-request concurrency wall (503s); Redis vs. router lifecycle mistake on pod restart.
- **Akshay:** mis-typed `Pending` counter (lifetime counter labeled as gauge); UI showing `is_repeat` instead of real Redis hits; AWS SSO / IAM / `imagePullPolicy` operational complexity.

Each ties back to course concepts: observability, back-pressure, flow control, cache coherence, Prometheus metric types (counter vs. gauge), stateful vs. stateless component lifecycles.

**Gaps:** None. Each member submits their own file separately on Canvas.

---

## Everyone-submits replica reminder

The assignment says: *"please make sure EACH team member uploads it under their own submission on Canvas."* Both Ajin and Akshay must upload items 1–5 to Canvas separately, even though the content is shared. Item 6 is each person's own file.

## Pre-submission actions still needed

1. Replace 3 placeholders in `piazza_community_contributions.md` (video URL, PDF attachment, CSV attachment).
2. Push `feature/ui-streamlit` → `main` so the default branch reflects the final state.
3. Verify the SharePoint video link opens for people outside the Northeastern tenant (or set it to "Anyone with link").
4. Upload all artifacts to Canvas — each team member submits individually.
5. Post `piazza_community_contributions.md` on Piazza under "Final Projects" with the video and PDF attached.
