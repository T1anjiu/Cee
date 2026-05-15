export interface ChatToken {
  type: 'text' | 'link'
  value: string
}

const URL_REGEX = /https?:\/\/[A-Za-z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+/g

const SAFE_CHARS = /^[A-Za-z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+$/

export function tokenizeChatMessage(text: string): ChatToken[] {
  const tokens: ChatToken[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = URL_REGEX.exec(text)) !== null) {
    if (match.index > lastIndex) {
      tokens.push({ type: 'text', value: text.slice(lastIndex, match.index) })
    }

    const url = match[0]
    // Validate URL with constructor
    try {
      const parsed = new URL(url)
      if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
        tokens.push({ type: 'link', value: url })
      } else {
        tokens.push({ type: 'text', value: url })
      }
    } catch {
      tokens.push({ type: 'text', value: url })
    }

    lastIndex = match.index + match[0].length
  }

  if (lastIndex < text.length) {
    tokens.push({ type: 'text', value: text.slice(lastIndex) })
  }

  return tokens
}

export function isValidUrlChar(char: string): boolean {
  return SAFE_CHARS.test(char)
}
