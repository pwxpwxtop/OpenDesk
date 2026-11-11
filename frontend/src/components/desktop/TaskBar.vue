<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useDesktopStore } from '../../stores/desktop'
import { APP_META, DESKTOP_ICONS } from '../../config/apps'

const props = defineProps({
  host: { type: String, default: '' },
  connected: { type: Boolean, default: true },
})

const emit = defineEmits(['disconnect'])

const desktop = useDesktopStore()
const now = ref(new Date())
let timer = null

const apps = DESKTOP_ICONS.map((key) => ({ key, ...APP_META[key] }))

const clock = computed(() => now.value.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))
const date = computed(() => now.value.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', weekday: 'short' }))

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})

function openApp(key) {
  desktop.startMenuOpen = false
  desktop.openWindow(key)
}

function onWindowClick(win) {
  if (desktop.activeId === win.id && !win.minimized) {
    desktop.minimize(win.id)
    return
  }
  desktop.focus(win.id)
}

function disconnect() {
  desktop.startMenuOpen = false
  emit('disconnect')
}
</script>

<template>
  <footer class="taskbar" @click.self="desktop.closeStartMenu()">
    <button type="button" class="taskbar-btn start" title="开始" @click.stop="desktop.toggleStartMenu()">
      <i class="fa-solid fa-table-cells-large"></i>
    </button>

    <div class="taskbar-divider"></div>

    <div class="taskbar-windows">
      <button
        v-for="win in desktop.windows"
        :key="win.id"
        type="button"
        class="taskbar-btn taskbar-item"
        :class="{ active: desktop.activeId === win.id && !win.minimized, running: true }"
        :title="win.title"
        @click="onWindowClick(win)"
      >
        <i :class="win.icon"></i>
        <span class="title">{{ win.title }}</span>
      </button>
    </div>

    <span class="od-badge" :class="{ online: connected }" :title="props.host">
      <i class="fa-solid fa-circle" style="font-size: 6px"></i>
      {{ connected ? host : '已断开' }}
    </span>

    <div class="taskbar-clock">
      <span class="time">{{ clock }}</span>
      <span>{{ date }}</span>
    </div>

    <div v-if="desktop.startMenuOpen" class="start-menu" @click.stop>
      <h4>应用</h4>
      <div class="start-apps">
        <button v-for="app in apps" :key="app.key" type="button" class="start-app" @click="openApp(app.key)">
          <span class="glyph" :style="{ '--icon-from': app.from, '--icon-to': app.to }">
            <i :class="app.icon"></i>
          </span>
          {{ app.title }}
        </button>
      </div>

      <h4>会话</h4>
      <button type="button" class="start-session" @click="disconnect">
        <i class="fa-solid fa-power-off" style="width: 14px; color: #f87171"></i>
        断开连接
      </button>
    </div>
  </footer>
</template>
