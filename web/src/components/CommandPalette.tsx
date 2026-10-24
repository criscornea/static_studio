import { useMemo, useState } from 'react'

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
  Command as CommandRoot,
} from '@/components/ui/command'
import { buildCommands, filterCommands } from '@/lib/commands'
import { formatShortcut } from '@/lib/shortcuts'
import { useUiStore } from '@/stores/ui'

/** Ctrl/Cmd+Shift+P: every command, filtered as you type. */
export function CommandPalette() {
  const open = useUiStore((s) => s.paletteOpen)
  const setOpen = useUiStore((s) => s.setPaletteOpen)

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Command palette"
      description="Search for a command to run"
    >
      {/* Mounted only while open, so the query and the command list start fresh. */}
      {open && <PaletteBody onDone={() => setOpen(false)} />}
    </CommandDialog>
  )
}

function PaletteBody({ onDone }: { onDone: () => void }) {
  const [query, setQuery] = useState('')
  const commands = useMemo(() => buildCommands(), [])
  const shown = filterCommands(commands, query)
  const groups = [...new Set(shown.map((c) => c.group))]

  return (
    // Filtering is ours (filterCommands), so cmdk only handles the keyboard.
    <CommandRoot shouldFilter={false} label="Commands">
      <CommandInput value={query} onValueChange={setQuery} placeholder="Type a command..." />
      <CommandList>
        <CommandEmpty>No command matches.</CommandEmpty>
        {groups.map((group) => (
          <CommandGroup key={group} heading={group}>
            {shown
              .filter((c) => c.group === group)
              .map((c) => (
                <CommandItem
                  key={c.id}
                  value={c.id}
                  disabled={!c.enabled}
                  onSelect={() => {
                    onDone()
                    c.run()
                  }}
                >
                  {c.title}
                  {c.shortcut && <CommandShortcut>{formatShortcut(c.shortcut)}</CommandShortcut>}
                </CommandItem>
              ))}
          </CommandGroup>
        ))}
      </CommandList>
    </CommandRoot>
  )
}
