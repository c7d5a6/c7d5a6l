const FLASH_CLASS = 'lamp-flash'
const FLASH_MS = 520

/**
 * Console confirm flash — bright lamp surge on any enabled button press.
 * Local to the pressed control only (no stage/viewport burst).
 */
export function installLampFlash(root: ParentNode = document): () => void {
  const onPointerDown = (event: Event) => {
    const target = event.target
    if (!(target instanceof Element)) return

    const btn = target.closest('button')
    if (!(btn instanceof HTMLButtonElement) || btn.disabled) return

    // Prefer pointerdown so the flash starts before click handlers unmount UI.
    flashButton(btn)
  }

  root.addEventListener('pointerdown', onPointerDown, true)
  return () => root.removeEventListener('pointerdown', onPointerDown, true)
}

function flashButton(btn: HTMLButtonElement) {
  btn.classList.remove(FLASH_CLASS)
  void btn.offsetWidth
  btn.classList.add(FLASH_CLASS)
  window.setTimeout(() => btn.classList.remove(FLASH_CLASS), FLASH_MS)
}
