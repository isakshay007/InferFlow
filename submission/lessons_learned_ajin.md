# Lessons Learned — InferFlow
**Ajin Frank Justin** · CS 6650 Spring 2026

## The metric that was lying to me

The most embarrassing moment in this project was running a load test, seeing "3/5 cache hits" printed in the terminal, and almost putting it in the report as evidence that `kv_aware` was working. When I actually read the generator code I found this:

```python
"cache_hit": is_repeat  # request_id % repeat_factor == 0
```

The number had nothing to do with Redis. It was pure arithmetic. We were measuring our own assumption. I fixed it by adding an `X-Inferflow-Cache-Hit` response header to the router — but even that took two tries because the first rebuild didn't actually deploy. The pods were running the old image because `imagePullPolicy: IfNotPresent` was caching the previous version. The header was in the code, the tests passed, but the live cluster was ignoring all of it.

This connects directly to what the course says about observability in distributed systems. You can have correct code that is completely invisible at runtime. The gap between "the code is right" and "the system is doing what I think" is where most bugs live in distributed systems. I now treat every metric as suspect until I can trace it back to a specific line of code that actually runs in production.

## Concurrency assumptions

My first real load test fired 20 concurrent requests at three llama.cpp backends and got 503s on everything except `round_robin`. I spent time thinking the routing logic was broken. It wasn't — llama.cpp processes one request at a time. With 20 concurrent requests and a 10 second timeout, 17 requests were queuing and timing out.

The course covers back-pressure and flow control as fundamental distributed systems problems. I understood the theory. I didn't apply it when designing the load test. I assumed the backends could absorb whatever I threw at them. The fix — a semaphore limiting concurrent requests to match backend capacity — is textbook back-pressure. I just had to break things first to understand why it matters.

## Redis and the router are not the same thing

After running hundreds of requests I wanted a clean demo, so I restarted the router pod. The KV hit rate in the UI didn't change. I assumed restarting the router would reset everything. It reset the in-memory counters but not Redis — which still had every prompt hash from the past two days stored with backend mappings. So the "first" request after restart was immediately a cache hit, and the hit rate stayed at 70%+.

This is a classic distributed systems mistake: treating a stateless component (the router) and a stateful component (Redis) as one unit. They have independent lifecycles. Restarting the router doesn't touch Redis any more than restarting a web server empties its database. The fix was `redis-cli FLUSHALL` — one command, but it required understanding that the state lived somewhere else entirely.

## What I'd do differently

Start every experiment by asking: where does each number actually come from? Trace it from the metric on screen back to the line of code that produced it. If you can't do that, the number is not trustworthy.
