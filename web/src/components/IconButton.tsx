import type { ComponentProps } from 'react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface Props extends Omit<ComponentProps<typeof Button>, 'size' | 'children'> {
  /** Accessible name. */
  label: string
  /** Tooltip text when it should differ from the label. */
  tip?: string
  /** Shortcut hint shown in the tooltip, e.g. "Ctrl+J". */
  shortcut?: string
  children: React.ReactNode
}

/** A square icon button with a tooltip that names the action and its shortcut. */
export function IconButton({ label, tip, shortcut, children, variant = 'ghost', ...props }: Props) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button size="icon-sm" variant={variant} aria-label={label} {...props}>
            {children}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {tip ?? label}
          {shortcut && <span className="text-background/70">{shortcut}</span>}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
