<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  label: { type: String, required: true },
  icon: { type: String, required: true },
  from: { type: String, default: '#4f8cff' },
  to: { type: String, default: '#2563eb' },
  x: { type: Number, default: 16 },
  y: { type: Number, default: 16 },
  selected: { type: Boolean, default: false },
})

const emit = defineEmits(['open', 'select', 'move', 'moveend', 'menu'])

const dragging = ref(false)
let dragState = null
let ignoreClick = false

const style = computed(() => ({
  left: `${props.x}px`,
  top: `${props.y}px`,
}))

function onPointerDown(event) {
  if (event.button !== 0) return
  dragState = {
    startX: event.clientX,
    startY: event.clientY,
    originX: props.x,
    originY: props.y,
    moved: false,
  }
  emit('select')
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

function onPointerMove(event) {
  if (!dragState) return
  const dx = event.clientX - dragState.startX
  const dy = event.clientY - dragState.startY
  if (!dragState.moved && Math.abs(dx) < 4 && Math.abs(dy) < 4) return
  dragState.moved = true
  dragging.value = true
  emit('move', { x: dragState.originX + dx, y: dragState.originY + dy })
}

function onPointerUp(event) {
  const state = dragState
  detach()
  if (!state) return
  if (state.moved) {
    // 拖拽结束后不能被当作点击/双击处理。
    ignoreClick = true
    window.setTimeout(() => {
      ignoreClick = false
    }, 160)
    emit('moveend', {
      x: state.originX + event.clientX - state.startX,
      y: state.originY + event.clientY - state.startY,
    })
  }
}

function detach() {
  dragState = null
  dragging.value = false
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
}

function onClick() {
  if (ignoreClick) return
  emit('select')
}

function onDoubleClick() {
  if (ignoreClick) return
  emit('open')
}

onBeforeUnmount(detach)
</script>

<template>
  <div
    class="desktop-icon"
    :class="{ selected, dragging }"
    :style="style"
    :title="`${props.label}（双击打开，可拖动）`"
    @pointerdown="onPointerDown"
    @click="onClick"
    @dblclick="onDoubleClick"
    @contextmenu.prevent.stop="emit('menu', $event)"
  >
    <span class="glyph" :style="{ '--icon-from': from, '--icon-to': to }">
      <i :class="icon"></i>
    </span>
    <span class="label">{{ label }}</span>
  </div>
</template>
