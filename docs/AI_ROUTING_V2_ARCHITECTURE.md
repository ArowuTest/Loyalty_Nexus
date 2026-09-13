# Loyalty Nexus AI Routing V2 — Approved Architecture

Status: implementation baseline, 2026-09-03
Baseline commit: 7c65350adf2e4c62d94098d5ac1a972f26137008

## Architectural invariant

No AI tool may depend on a provider, model, endpoint, API key, or fallback order hardcoded in application business logic. Runtime execution is governed by Admin-managed routing configuration.

The routing unit is **tool + stage + provider/model candidate**, not a global category. Composite tools may have multiple independently routed stages.

A new model/provider using an already-supported protocol must be deploy-free. A new tool using an existing execution profile must be deploy-free. Only a genuinely new protocol or workflow primitive may require engineering.

## Runtime decision

For each request the router resolves:
1. tool and execution stage;
2. routing policy and eligible candidate bindings;
3. capability compatibility;
4. circuit/health state and current capacity;
5. cost tier and configured spending policy;
6. highest-ranked eligible candidate with capacity.

There is no hidden compiled fallback when no configured route exists. An unroutable stage is unavailable and must not silently use an engineer-selected provider.
## Core data model

- `studio_tools`: product definition, pricing, UI and execution profile.
- `ai_provider_configs`: provider/model candidate inventory and credentials.
- `ai_tool_stages`: executable stages for a tool, capability and routing policy.
- `ai_tool_provider_bindings`: ordered candidates per tool stage, including surge limits and cost tier.
- `ai_generation_attempts`: immutable per-attempt telemetry for audit, health and cost attribution.
- `ai_model_catalog`: discovered model metadata from OpenRouter/public sources.
- `ai_model_scores`: public, Nexus-eval and production scores by capability.

Priority is a property of the tool-stage binding, not globally of the provider.

## Cost and quality policies

Supported routing policies are initially:
- `FREE_FIRST`: prefer approved free candidates, then low-cost/premium only if policy permits.
- `BALANCED`: quality/reliability/latency/cost weighted.
- `QUALITY_FIRST`: use highest-quality approved candidates first.
- `FREE_ONLY`: never incur paid fallback cost.
- `PREMIUM_ONLY`: use only explicitly approved premium candidates.

User Pulse Point pricing remains owned by `studio_tools.point_cost`. Provider `cost_micros` is operational cost telemetry and must not independently charge the user.
## Surge and capacity control

Every binding may define:
- max concurrent requests;
- requests per minute;
- queue class;
- cost tier;
- whether paid fallback is allowed.

Capacity must be reserved before a provider call. Distributed capacity uses Redis so multiple API/worker instances share one authority. If one free model is saturated, the router may select another eligible free model without first generating a 429 stampede.

Traffic classes:
- REALTIME: chat and short interactive requests;
- INTERACTIVE: coding, research and document analysis;
- ASYNC: images, audio and avatars;
- HEAVY_ASYNC: video/composite generation;
- BACKGROUND: Model Scout and benchmark work, always lowest priority.

The router must prevent fallback stampedes and paid-cost cascades. Provider retries are attempts under one generation ID and never create additional customer charges.

## Model intelligence and Scout

A background Model Intelligence worker ingests OpenRouter/public model metadata and benchmark signals, stores snapshots locally, and shortlists promising models. A FREE_FIRST AI Model Scout may reason over the shortlist and recommend promotion/demotion.

The Scout never has unilateral production-control authority. Initial promotion is Admin-approved. Runtime routing always reads locally cached intelligence/configuration rather than querying public benchmarks on a customer request.
## Health, failover and governance

Provider errors are classified. Network/timeout/429/5xx/provider-malformed responses may fail over. Invalid user input and safety/policy refusals must not trigger provider shopping.

Provider health is stage-aware. A failed primary with a healthy backup means DEGRADED, not unavailable. Repeated provider failures open a circuit; cooldown leads to half-open probes and recovery.

Admin changes follow Edit → Validate/Test → Publish, with emergency Disable Now. Changes are auditable and rollbackable. Credentials are never returned to clients and production must refuse weak base64-only secret storage.

## Acceptance criteria

- every active AI tool has a valid execution profile and route;
- all provider selection flows through the central router;
- Nexus Chat uses the same routing authority;
- no Studio tool directly reads provider secrets or chooses vendor/model order;
- Admin route changes affect the next request without restart/deploy;
- surge limits are distributed and provider/model specific;
- no provider capacity causes a thundering-herd fallback;
- no route before charging, or exact-once refund after failure;
- every provider attempt is auditable with model, outcome, latency and cost;
- strong credential encryption is mandatory in production;
- architecture tests prevent reintroduction of hardcoded provider routing.

## Delivery sequence

1. schema + repository + route-resolution tests;
2. distributed capacity controller + attempt ledger;
3. migrate standard Studio dispatch to Router V2;
4. migrate composite/specialised stages;
5. converge Nexus Chat on Router V2;
6. Model Intelligence/Scout ingestion and Admin recommendations;
7. Admin tool-routing UX, publish/rollback and health dashboards;
8. parity/failure/surge/security gates; remove legacy hardcoded routing only after all active tools are covered.
