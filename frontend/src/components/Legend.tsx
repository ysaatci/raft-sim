const item = 'flex items-center gap-1.5'

/** Explains the packet encoding used by Packets. */
export function Legend() {
  return (
    <ul
      aria-label="Legend"
      className="flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground"
    >
      <li className={item}>
        <span className="size-2.5 rounded-full bg-msg-vote" /> RequestVote
      </li>
      <li className={item}>
        <span className="size-2.5 rounded-full bg-msg-append" /> AppendEntries
      </li>
      <li className={item}>
        <span className="size-2.5 rounded-full border-2 border-msg-append" /> response
      </li>
      <li className={item}>
        <span className="size-2.5 rounded-full border-2 border-crashed" /> rejected
      </li>
      <li className={item}>
        <span className="size-2.5 rounded-full bg-msg-append opacity-40" /> dropped
      </li>
    </ul>
  )
}
