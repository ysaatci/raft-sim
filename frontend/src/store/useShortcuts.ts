import { useEffect } from 'react'
import { simStore } from './sim'

const STEP_MS = 10

/**
 * Space: play/pause · →: step · 1–9: select a node (or pick it for a
 * partition) · Esc: deselect and cancel a partition.
 */
export function useShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey || e.altKey) return
      const t = e.target as HTMLElement
      if (t.closest('input, select, textarea, [contenteditable=true]')) return
      // Let focused buttons (nodes, links, controls) handle their own keys.
      if ((e.key === ' ' || e.key === 'Enter') && t.closest('button, [role=button], [role=slider]')) return

      const s = simStore.getState()
      if (e.key === ' ') {
        e.preventDefault()
        if (s.playing) s.pause()
        else s.play()
      } else if (e.key === 'ArrowRight' && !s.playing && !t.closest('[role=slider]')) {
        void s.step(STEP_MS)
      } else if (e.key === 'Escape') {
        s.cancelPartition()
        s.select(null)
      } else if (/^[1-9]$/.test(e.key) && s.state && Number(e.key) <= s.state.nodes.length) {
        const id = Number(e.key)
        if (s.partitionDraft) s.togglePartitionNode(id)
        else s.select(s.selected === id ? null : id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
}
