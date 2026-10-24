import type { LucideIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon: LucideIcon
  /** One line that says what is missing. */
  text: string
  action?: { label: string; onClick: () => void }
  className?: string
}

/** An icon, one line of text and the action that fixes it. */
export function EmptyState({ icon: Icon, text, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-2 px-4 py-6 text-center text-sm text-muted-foreground',
        className,
      )}
    >
      <Icon className="size-6 opacity-60" aria-hidden />
      <p>{text}</p>
      {action && (
        <Button variant="outline" size="sm" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  )
}

const widths = ['w-3/4', 'w-1/2', 'w-2/3', 'w-5/6', 'w-1/3']

/** Placeholder lines shown while something loads. */
export function SkeletonRows({
  count = 5,
  label = 'Loading',
  className,
}: {
  count?: number
  label?: string
  className?: string
}) {
  return (
    <div role="status" aria-busy="true" className={cn('space-y-2 p-3', className)}>
      <span className="sr-only">{label}</span>
      {Array.from({ length: count }, (_, i) => (
        <Skeleton key={i} className={cn('h-4', widths[i % widths.length])} />
      ))}
    </div>
  )
}
