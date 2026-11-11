<script setup>
import { computed } from 'vue'
import { useDesktopStore } from '../../stores/desktop'

const props = defineProps({
  win: { type: Object, required: true },
})

const store = useDesktopStore()
const resizeHandles = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']

const style = computed(() => ({
  left: `${props.win.x}px`,
  top: `${props.win.y}px`,
  width: `${props.win.width}px`,
  height: `${props.win.height}px`,
  zIndex: props.win.z,
  display: props.win.minimized ? 'none' : 'flex',
}))

function startDrag(event) {
  if (event.button !== 0 || props.win.maximized) return
  store.focus(props.win.id)

  const startX = event.clientX
  const startY = event.clientY
  const originX = props.win.x
  const originY = props.win.y
  const winId = props.win.id

  const onMove = (moveEvent) => {
    store.setBounds(winId, {
      x: originX + moveEvent.clientX - startX,
      y: originY + moveEvent.clientY - startY,
    })
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

function startResize(event, handle) {
  if (event.button !== 0 || props.win.maximized) return
  event.preventDefault()
  store.focus(props.win.id)

  const startX = event.clientX
  const startY = event.clientY
  const origin = { x: props.win.x, y: props.win.y, width: props.win.width, height: props.win.height }
  const winId = props.win.id

  const onMove = (moveEvent) => {
    const dx = moveEvent.clientX - startX
    const dy = moveEvent.clientY - startY
    const next = { ...origin }

    if (handle.includes('e')) next.width = origin.width + dx
    if (handle.includes('s')) next.height = origin.height + dy
    if (handle.includes('w')) {
      next.width = origin.width - dx
      next.x = origin.x + dx
    }
    if (handle.includes('n')) {
      next.height = origin.height - dy
      next.y = origin.y + dy
    }
    store.setBounds(winId, next)
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}
</script>

<template>
  <section
    class="window"
    :class="{ active: store.activeId === win.id, maximized: win.maximized }"
    :style="style"
    @pointerdown.capture="store.focus(win.id)"
  >
    <header class="win-titlebar" @pointerdown.stop="startDrag" @dblclick="store.toggleMaximize(win.id)">
      <span class="win-icon"><i :class="win.icon"></i></span>
      <span class="win-title">{{ win.title }}</span>
      <span class="win-actions" @pointerdown.stop @dblclick.stop>
        <button type="button" title="最小化" @click="store.minimize(win.id)">
          <i class="fa-solid fa-minus"></i>
        </button>
        <button type="button" :title="win.maximized ? '还原' : '最大化'" @click="store.toggleMaximize(win.id)">
          <i :class="win.maximized ? 'fa-regular fa-clone' : 'fa-regular fa-square'"></i>
        </button>
        <button type="button" class="close" title="关闭" @click="store.close(win.id)">
          <i class="fa-solid fa-xmark"></i>
        </button>
      </span>
    </header>

    <div class="win-body">
      <slot />
    </div>

    <div
      v-for="handle in resizeHandles"
      :key="handle"
      class="resize-handle"
      :class="`rh-${handle}`"
      @pointerdown.stop="startResize($event, handle)"
    ></div>
  </section>
</template>
