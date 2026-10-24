import { Monitor, Moon, Sun } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useThemeStore, type ThemeChoice } from '@/stores/theme'

const next: Record<ThemeChoice, ThemeChoice> = { system: 'light', light: 'dark', dark: 'system' }
const icons = { system: Monitor, light: Sun, dark: Moon } as const

/** Cycles the theme through system, light and dark. */
export function ThemeButton() {
  const choice = useThemeStore((s) => s.choice)
  const setChoice = useThemeStore((s) => s.setChoice)
  const Icon = icons[choice]

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label={`Theme: ${choice}. Switch to ${next[choice]}`}
      title={`Theme: ${choice}`}
      onClick={() => setChoice(next[choice])}
    >
      <Icon />
    </Button>
  )
}
