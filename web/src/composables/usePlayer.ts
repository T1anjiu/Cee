export interface SyncResult {
  tier: number // 0=no adjustment, 1=±5%, 2=±10%, 3=seek
  rate: number
  shouldSeek: boolean
}

const ENTER = [0.1, 0.3, 1.0]
const EXIT = [0.05, 0.25, 0.95]

export function computeSync(diff: number, prevTier: number): SyncResult {
  const abs = Math.abs(diff)
  let tier = prevTier

  if (abs >= ENTER[tier]) {
    tier++
    if (tier > 3) tier = 3
  } else if (tier > 0 && abs <= EXIT[tier - 1]) {
    tier--
  }

  const sign = diff > 0 ? 1 : -1
  let rate = 1.0
  let shouldSeek = false

  switch (tier) {
    case 0:
      rate = 1.0
      break
    case 1:
      rate = sign > 0 ? 0.95 : 1.05
      break
    case 2:
      rate = sign > 0 ? 0.90 : 1.10
      break
    case 3:
      shouldSeek = true
      rate = 1.0
      break
  }

  return { tier, rate, shouldSeek }
}

export function usePlayer() {
  let currentTier = 0

  function compute(diff: number): SyncResult {
    const result = computeSync(diff, currentTier)
    currentTier = result.tier
    return result
  }

  function reset() {
    currentTier = 0
  }

  return { compute, reset }
}
