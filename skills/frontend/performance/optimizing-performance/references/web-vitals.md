# Web Vitals

## Core Metrics

| Metric | Target | Measures                   |
| ------ | ------ | -------------------------- |
| LCP    | <2.5s  | Largest content paint time |
| INP    | <=200ms | Interaction responsiveness |
| CLS    | <0.1   | Cumulative layout shift    |

## LCP Optimization

| Technique      | Implementation                          |
| -------------- | --------------------------------------- |
| Priority load  | `fetchpriority="high"` on hero image    |
| Preload        | `<link rel="preload" as="image">`       |
| No lazy on LCP | Remove `loading="lazy"` from above-fold |
| Code splitting | `lazy()` for non-critical routes        |

## CLS Prevention

| Cause               | Solution                               |
| ------------------- | -------------------------------------- |
| Images without size | Add `width`/`height` or `aspect-ratio` |
| Dynamic content     | Reserve space with `min-height`        |
| Web fonts           | `font-display: swap`                   |
| Ads/embeds          | Fixed containers                       |

## INP Optimization

| Technique          | Impact                    |
| ------------------ | ------------------------- |
| Code splitting     | Reduce initial JS         |
| Long tasks         | Break into <50ms chunks   |
| Web workers        | Offload heavy computation |
| Defer non-critical | `defer`/`async` scripts   |

## Measurement

```typescript
import { onCLS, onINP, onLCP } from "web-vitals";

onCLS(console.log);
onINP(console.log);
onLCP(console.log);
```

## Checklist

| Check            | Target                         |
| ---------------- | ------------------------------ |
| Lighthouse score | ≥90                            |
| LCP element      | Priority loaded                |
| Images           | Sized, WebP, lazy (below fold) |
| Dynamic content  | Space reserved                 |

## Current measurement guidance

Use LCP, INP and CLS at the 75th percentile, split by mobile and desktop.
A Lighthouse navigation run cannot measure real-user INP; distinguish lab
results and field measurements. Reference: https://web.dev/articles/vitals
