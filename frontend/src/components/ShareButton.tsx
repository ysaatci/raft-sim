import { Check, Link2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { useSim } from '@/store/sim'

/** Copies a link that reproduces this exact run at this moment. */
export function ShareButton() {
  const shareLink = useSim((s) => s.shareLink)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const t = setTimeout(() => setCopied(false), 2000)
    return () => clearTimeout(t)
  }, [copied])

  const share = async () => {
    const fragment = await shareLink()
    if (!fragment) return
    history.replaceState(null, '', `#${fragment}`) // the address bar holds the link too
    try {
      await navigator.clipboard.writeText(location.href)
    } catch {
      // No clipboard access (e.g. insecure context): the address bar has it.
    }
    setCopied(true)
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={share}
      aria-label={copied ? 'Link copied' : 'Copy link to this moment'}
    >
      {copied ? <Check /> : <Link2 />}
      {copied ? 'Copied' : 'Share'}
    </Button>
  )
}
