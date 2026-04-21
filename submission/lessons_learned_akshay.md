# Lessons Learned — InferFlow
**Akshay Keerthi Adhikasavan Suresh** · CS 6650 Spring 2026

## The pending count that wasn't pending

While working on the Streamlit UI I noticed the live metrics panel was showing backend-1 with 177 pending requests, backend-2 with 188, backend-3 with 166 — while in-flight was showing 0. Those numbers can't both be right. Either requests are in flight or they aren't.

I traced it back to the `/api/status` handler. The `Pending` field was being populated from `backendSelections[b.Name]` — the total number of requests ever sent to that backend, not the current in-flight count. The label said "pending" but the value was a lifetime counter.

This is a naming and contract problem. In a distributed system every metric needs a precise definition: is this a counter (ever-increasing), a gauge (current state), or a rate? The course covers this in the context of Prometheus metric types. I knew the theory. I just didn't apply it when reading someone else's code. I should have checked the type before building UI on top of it. A metric with the wrong label is more dangerous than a missing metric — it gives you false confidence.

## The UI was showing what I wanted to see

When I first tested the Streamlit UI with `kv_aware` strategy, the routing decision log showed cache hits immediately. I thought it was working perfectly. It was — but for the wrong reason. The `cache_hit` value being displayed was coming from `is_repeat` in the load generator, not from the router. Every third request was flagged as a hit regardless of what Redis actually said.

The real hit rate only became visible after we added the `X-Inferflow-Cache-Hit` header to the router response and updated the generator to read it. Before that fix, the UI was displaying a number that felt right but was completely disconnected from the system's actual behavior.

The course talks about the difference between measuring a system and measuring your assumptions about a system. This was a live example. The UI looked correct because I had built it to show what I expected to see. Validating that the display matched reality required going all the way down to the Redis lookup in the router — not just trusting the number that appeared on screen.

## AWS credentials and operational reality

Something the course doesn't cover but that dominated hours of this project: session credentials expire. Every time I came back to test something, the kubectl context was broken. AWS SSO tokens have a TTL. The cluster is real infrastructure, not a local process you can just restart.

This taught me something about operational complexity that's hard to appreciate from a textbook. In production, the gap between "the code works" and "I can actually access the running system" is filled with credential management, IAM policies, VPC configurations, and load balancer rules. We spent real time on ALB IAM policy errors, missing `DescribeListenerAttributes` permissions, `imagePullPolicy` misconfiguration. Each one was a one-line fix that took an hour to diagnose.

The course concept that applies here is the difference between a system's logical design and its operational reality. We designed a clean routing system. Operating it on real AWS infrastructure introduced a whole second layer of complexity that had nothing to do with routing algorithms.

## What I'd do differently

Write down what every metric means before building UI on top of it. Is it a counter or a gauge? What resets it? What increments it? If you can't answer those questions, the display is fiction.
