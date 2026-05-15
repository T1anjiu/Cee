export function normalizeRoomCode(input: string): string {
  let s = input.trim().toUpperCase()
  s = s.replace(/O/g, '0')
  s = s.replace(/I/g, '1')
  s = s.replace(/L/g, '1')
  return s
}
