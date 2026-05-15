const UA = navigator.userAgent

export function isMobileDevice(): boolean {
  return (
    /Android|iPhone|iPod|Windows Phone|HarmonyOS/i.test(UA) ||
    /iPad/i.test(UA) ||
    (/Macintosh/i.test(UA) && navigator.maxTouchPoints > 1)
  )
}

export function getMobileOverride(): boolean {
  const val = sessionStorage.getItem('mobile_override')
  if (val === 'force_mobile') return true
  if (val === 'force_desktop') return false
  return isMobileDevice()
}

export function setMobileOverride(forceMobile: boolean) {
  sessionStorage.setItem('mobile_override', forceMobile ? 'force_mobile' : 'force_desktop')
}
