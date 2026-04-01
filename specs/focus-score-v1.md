# Focus Score — Specification v1

## Purpose

A single daily metric (0.0–1.0) that captures how much of your tracked time was spent in sustained, deep focus. The score rewards long uninterrupted blocks of deep work and penalizes fragmentation and shallow-heavy days.

## Definitions

### Focus Block

A **focus block** is one or more consecutive same-topic sessions where:

1. No other-topic session appears between them (chronologically).
2. No intra-topic gap exceeds **15 minutes**.

If either condition is violated, a new block begins.

### Block Score

```
block_score = min(block_duration / 90, 1.0)
```

90 minutes is the gold standard. Blocks shorter than 90m score proportionally. Blocks longer than 90m cap at 1.0.

### Topic Weight

Each topic is classified by the user at log time:

| Type     | Weight |
|----------|--------|
| Deep     | 1.0    |
| Shallow  | 0.5    |

Deep = engineering, design, writing, research, technical review.
Shallow = meetings, admin, status updates, Slack triage, capacity planning.

## Formula

```
Focus Score = Σ(block_score_i × duration_i × weight_i) / Σ(duration_i)
```

- Numerator: weighted contribution of each block.
- Denominator: total tracked time (raw, unweighted).

This means:
- A perfect day of four 90m deep blocks → **1.0**
- A perfect day of four 90m shallow blocks → **0.5**
- A day of thirty 10m deep fragments → **~0.11**
- A day of thirty 10m shallow fragments → **~0.06**

## Parameters

| Parameter       | Value | Rationale                                    |
|-----------------|-------|----------------------------------------------|
| Target duration | 90m   | Ultradian rhythm upper bound (goal to train toward) |
| Max gap         | 15m   | Coffee/bathroom break tolerance              |
| Deep weight     | 1.0   | Full contribution                            |
| Shallow weight  | 0.5   | Tracked but de-prioritized                   |
| Score cap       | 1.0   | No flow-state bonus in v1                    |

## Worked Example: Monday 2026-03-31

### Raw log → chronological events

| Time  | Topic           | Duration | Type    |
|-------|-----------------|----------|---------|
| 09:41 | Firewall Hits   | 18m      | deep    |
| 10:01 | Plan Review     | 37m      | shallow |
| 10:39 | Firewall Hits   | 3m       | deep    |
| 10:58 | Firewall Hits   | 10m      | deep    |
| 11:14 | Firewall Hits   | 49m      | deep    |
| 12:04 | Staff           | 4m       | shallow |
| 13:20 | Firewall Hits   | 9m       | deep    |
| 13:30 | Staff           | 5m       | shallow |
| 13:35 | Operations      | 12m      | deep    |
| 13:48 | Staff           | 4m       | shallow |
| 13:56 | Sessiondb       | 6m       | deep    |
| 14:03 | Admin Misc      | 39m      | shallow |
| 14:57 | Sessiondb       | 10m      | deep    |
| 15:09 | Admin Misc      | ~5m      | shallow |
| 15:14 | Admin Misc      | 25m      | shallow |
| 15:50 | Adr8            | 10m      | deep    |
| 16:01 | Adr8            | 100m     | deep    |

### Block construction

**Block 1 — Firewall Hits, 18m (deep)**
09:41–09:59. Next FH at 10:39, but Plan Review (10:01) intervenes → block ends.

**Block 2 — Plan Review, 37m (shallow)**
10:01–10:38. Only session.

**Block 3 — Firewall Hits, 3m (deep)**
10:39–10:42. Next FH at 10:58, no other topic intervenes, but gap = 16m > 15m → block ends.

**Block 4 — Firewall Hits, 59m (deep)**
10:58–11:08 (10m) + 11:14–12:03 (49m). Gap = 6m < 15m, no intervening topic → merged.

**Block 5 — Staff, 4m (shallow)**
12:04–12:08. Next Staff at 13:30, but FH (13:20) intervenes → block ends.

**Block 6 — Firewall Hits, 9m (deep)**
13:20–13:29. No subsequent FH sessions.

**Block 7 — Staff, 5m (shallow)**
13:30–13:35. Next Staff at 13:48, but Operations (13:35) intervenes → block ends.

**Block 8 — Operations, 12m (deep)**
13:35–13:47. Only session.

**Block 9 — Staff, 4m (shallow)**
13:48–13:52. No subsequent Staff sessions.

**Block 10 — Sessiondb, 6m (deep)**
13:56–14:02. Next Sessiondb at 14:57, but Admin Misc (14:03) intervenes → block ends.

**Block 11 — Admin Misc, 39m (shallow)**
14:03–14:42. Next Admin Misc at 15:09, but Sessiondb (14:57) intervenes → block ends.

**Block 12 — Sessiondb, 10m (deep)**
14:57–15:07. No subsequent Sessiondb sessions.

**Block 13 — Admin Misc, 30m (shallow)**
15:09–15:14 (~5m) + 15:14–15:39 (25m). No intervening topic, gap < 15m → merged.

**Block 14 — Adr8, 110m (deep)**
15:50–16:00 (10m) + 16:01–17:41 (100m). Gap = 1m, no intervening topic → merged.

### Score calculation

| #  | Topic         | Dur  | Weight | block_score | Contribution (score × dur × weight) |
|----|---------------|------|--------|-------------|--------------------------------------|
| 1  | Firewall Hits | 18m  | 1.0    | 0.200       | 3.60                                 |
| 2  | Plan Review   | 37m  | 0.5    | 0.411       | 7.60                                 |
| 3  | Firewall Hits | 3m   | 1.0    | 0.033       | 0.10                                 |
| 4  | Firewall Hits | 59m  | 1.0    | 0.656       | 38.69                                |
| 5  | Staff         | 4m   | 0.5    | 0.044       | 0.09                                 |
| 6  | Firewall Hits | 9m   | 1.0    | 0.100       | 0.90                                 |
| 7  | Staff         | 5m   | 0.5    | 0.056       | 0.14                                 |
| 8  | Operations    | 12m  | 1.0    | 0.133       | 1.60                                 |
| 9  | Staff         | 4m   | 0.5    | 0.044       | 0.09                                 |
| 10 | Sessiondb     | 6m   | 1.0    | 0.067       | 0.40                                 |
| 11 | Admin Misc    | 39m  | 0.5    | 0.433       | 8.45                                 |
| 12 | Sessiondb     | 10m  | 1.0    | 0.111       | 1.11                                 |
| 13 | Admin Misc    | 30m  | 0.5    | 0.333       | 5.00                                 |
| 14 | Adr8          | 110m | 1.0    | 1.000       | 110.00                               |

**Numerator**: 177.77
**Denominator**: 346m (total tracked time)

### **Monday Focus Score: 0.51**

### Interpretation

The score is honest. The Adr8 block (110m, deep) is doing most of the heavy lifting — contributing 110 of 177.77 to the numerator, or 62% of the score from 32% of the time. The afternoon interleaving (Staff/Ops/Sessiondb/Admin cycling between 13:20–15:07) contributes almost nothing. Block 3 (a lone 3m FH session orphaned by a 16m gap) shows how the 15m threshold works in practice.

## Edge Cases

### Zero tracked time
Score is undefined. Display as `—` or `N/A`.

### All shallow day
Caps at 0.5. This is intentional — a day of pure meetings, no matter how well structured, is not a focused day.

### Single block exceeding 90m
Caps at 1.0. No flow-state bonus in v1.

### Overnight / multi-day
Score is per calendar day. Sessions crossing midnight split at 00:00.

### Unlogged gaps
Time not tracked does not appear in the denominator. The score measures quality of tracked time, not total work time. A 2h tracked day with one perfect 90m deep block scores higher than an 8h tracked day of fragments. This is a feature: it rewards intentional tracking and focus, not volume.
