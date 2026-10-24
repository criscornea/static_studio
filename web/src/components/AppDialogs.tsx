import { CommandPalette } from '@/components/CommandPalette'
import { DeletePageDialog } from '@/components/DeletePageDialog'
import { NewPageDialog } from '@/components/NewPageDialog'
import { PublishDialog } from '@/components/PublishDialog'
import { QuickOpen } from '@/components/QuickOpen'
import { RenamePageDialog } from '@/components/RenamePageDialog'
import { ShortcutsDialog } from '@/components/ShortcutsDialog'
import { useUiStore } from '@/stores/ui'

/** Dialogs that commands can open, mounted once so they work with the sidebar hidden. */
export function AppDialogs() {
  const newPage = useUiStore((s) => s.newPage)
  const closeNewPage = useUiStore((s) => s.closeNewPage)
  const renamePath = useUiStore((s) => s.renamePath)
  const closeRename = useUiStore((s) => s.closeRename)
  const deletePath = useUiStore((s) => s.deletePath)
  const closeDelete = useUiStore((s) => s.closeDelete)

  return (
    <>
      <NewPageDialog
        open={newPage !== null}
        onOpenChange={(open) => !open && closeNewPage()}
        folder={newPage?.folder}
      />
      <RenamePageDialog path={renamePath} onClose={closeRename} />
      <DeletePageDialog path={deletePath} onClose={closeDelete} />
      <PublishDialog />
      <CommandPalette />
      <QuickOpen />
      <ShortcutsDialog />
    </>
  )
}
