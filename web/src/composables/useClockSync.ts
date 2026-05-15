import { ref } from 'vue'
import type { Message } from '../types/message'

export function useClockSync() {
  const offset = ref(0)
  const rtt = ref(0)
  const samples = ref<{ offset: number; rtt: number }[]>([])
  let sampleCount = 0
  const targetSamples = 5

  function handlePong(msg: Message) {
    const payload = msg.payload as unknown as { t1: number; t2: number }
    const t3 = Date.now()
    const r = t3 - payload.t1
    const o = ((payload.t2 - payload.t1) + (payload.t2 - t3)) / 2
    samples.value.push({ offset: o, rtt: r })
    sampleCount++

    if (samples.value.length === 1 || r < rtt.value) {
      offset.value = o
      rtt.value = r
    }
  }

  function sendPing(send: (msg: Message) => void) {
    if (sampleCount < targetSamples) {
      send({
        type: 'ping',
        payload: { t1: Date.now() },
      })
    }
  }

  function serverNow(): number {
    return Date.now() + offset.value
  }

  function reset() {
    sampleCount = 0
    samples.value = []
    offset.value = 0
    rtt.value = 0
  }

  return {
    offset,
    rtt,
    sampleCount,
    targetSamples,
    handlePong,
    sendPing,
    serverNow,
    reset,
  }
}
