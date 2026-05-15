import { describe, it, expect } from 'vitest'
import { computeSync } from '../usePlayer'

describe('sync tier computation', () => {
  it('returns tier 0 for diff < 0.1', () => {
    const r = computeSync(0.05, 0)
    expect(r.tier).toBe(0)
    expect(r.rate).toBe(1)
  })

  it('enters tier 1 for diff >= 0.1', () => {
    const r = computeSync(0.12, 0)
    expect(r.tier).toBe(1)
    expect(r.rate).toBe(0.95)
  })

  it('hysteresis: stays in tier 1 when diff drops to 0.08', () => {
    const r = computeSync(0.08, 1)
    expect(r.tier).toBe(1)
    expect(r.rate).toBe(0.95)
  })

  it('hysteresis: exits tier 1 when diff <= 0.05', () => {
    const r = computeSync(0.04, 1)
    expect(r.tier).toBe(0)
    expect(r.rate).toBe(1)
  })

  it('enters tier 2 for diff >= 0.3 from tier 1', () => {
    const r = computeSync(0.32, 1)
    expect(r.tier).toBe(2)
    expect(r.rate).toBe(0.90)
  })

  it('steps up to tier 3 (seek) across cycles for diff >= 1.0', () => {
    let tier = 0
    let r = computeSync(1.05, tier)
    expect(r.tier).toBe(1)
    tier = r.tier
    r = computeSync(1.05, tier)
    expect(r.tier).toBe(2)
    tier = r.tier
    r = computeSync(1.05, tier)
    expect(r.tier).toBe(3)
    expect(r.shouldSeek).toBe(true)
    expect(r.rate).toBe(1)
  })

  it('handles negative diff (ahead)', () => {
    const r = computeSync(-0.15, 0)
    expect(r.tier).toBe(1)
    expect(r.rate).toBe(1.05)
  })

  it('handles large negative diff across cycles', () => {
    let tier = 0
    let r = computeSync(-0.35, tier)
    expect(r.tier).toBe(1)
    r = computeSync(-0.35, r.tier)
    expect(r.tier).toBe(2)
    expect(r.rate).toBe(1.10)
  })

  it('negative diff seek via multi-cycle', () => {
    let tier = 0
    let r: ReturnType<typeof computeSync> | undefined
    for (let i = 0; i < 3; i++) {
      r = computeSync(-1.2, tier)
      tier = r.tier
    }
    expect(r!.tier).toBe(3)
    expect(r!.shouldSeek).toBe(true)
  })
})
