<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import Button from 'primevue/button'
import { attachOutput, onSessionClosed, SSHService, disconnect } from '../../services/ssh'

const props = defineProps({
  sessionId: { type: String, default: '' },
  host: { type: String, default: '' },
})

const emit = defineEmits(['open-app'])

const hostEl = ref(null)
const connected = ref(true)

let term = null
let fitAddon = null
let resizeObserver = null
let detachOutput = null
let detachClosed = null

onMounted(() => {
  if (!props.sessionId || !hostEl.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    lineHeight: 1.25,
    fontFamily: "'Cascadia Mono', 'JetBrains Mono', Consolas, 'Courier New', monospace",
    scrollback: 8000,
    theme: {
      background: '#0b1017',
      foreground: '#d7e2f4',
      cursor: '#4f8cff',
      cursorAccent: '#0b1017',
      selectionBackground: 'rgba(79, 140, 255, 0.35)',
      black: '#1b2433',
      red: '#f87171',
      green: '#34d399',
      yellow: '#fbbf24',
      blue: '#60a5fa',
      magenta: '#c084fc',
      cyan: '#22d3ee',
      white: '#e2e8f0',
      brightBlack: '#64748b',
      brightRed: '#fca5a5',
      brightGreen: '#6ee7b7',
      brightYellow: '#fcd34d',
      brightBlue: '#93c5fd',
      brightMagenta: '#d8b4fe',
      brightCyan: '#67e8f9',
      brightWhite: '#f8fafc',
    },
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(hostEl.value)

  const fit = () => {
    try {
      fitAddon.fit()
    } catch (error) {
      // The window may be hidden or not laid out yet.
    }
  }

  requestAnimationFrame(() => {
    fit()
    term.focus()
  })

  detachOutput = attachOutput(props.sessionId, (bytes) => {
    if (term) term.write(bytes)
  })

  detachClosed = onSessionClosed((closedId, reason) => {
    if (closedId !== props.sessionId || !term) return
    connected.value = false
    term.writeln(`\r\n\x1b[38;5;203m[连接已关闭] ${reason ?? ''}\x1b[0m`)
  })

  term.onData((data) => {
    SSHService.Write(props.sessionId, data).catch(() => {})
  })

  term.onResize(({ cols, rows }) => {
    SSHService.Resize(props.sessionId, cols, rows).catch(() => {})
  })

  resizeObserver = new ResizeObserver(() => fit())
  resizeObserver.observe(hostEl.value)
})

onBeforeUnmount(() => {
  detachOutput?.()
  detachClosed?.()
  resizeObserver?.disconnect()
  term?.dispose()
  term = null
})

function focusTerminal() {
  term?.focus()
}

function clearScreen() {
  term?.clear()
}

function openApp(kind) {
  emit('open-app', kind)
}

async function closeSession() {
  await disconnect(props.sessionId).catch(() => {})
}
</script>

<template>
  <div class="app-root">
    <div v-if="!sessionId" class="app-empty">
      <span class="glyph"><i class="fa-solid fa-terminal"></i></span>
      <div>这个窗口没有关联的 SSH 会话。</div>
    </div>

    <template v-else>
      <div class="app-header">
        <span class="od-badge" :class="{ online: connected }">
          <i class="fa-solid fa-circle" style="font-size: 6px"></i>
          {{ connected ? '已连接' : '已断开' }}
        </span>
        <span class="mono muted" style="font-size: 12px">{{ host }}</span>
        <span class="spacer"></span>
        <Button label="文件" icon="pi pi-folder" size="small" text severity="secondary" @click="openApp('files')" />
        <Button label="监控" icon="pi pi-chart-line" size="small" text severity="secondary" @click="openApp('monitor')" />
        <Button label="清屏" icon="pi pi-eraser" size="small" text severity="secondary" @click="clearScreen" />
        <Button label="断开" icon="pi pi-power-off" size="small" text severity="danger" @click="closeSession" />
      </div>

      <div class="terminal-wrap" @click="focusTerminal">
        <div ref="hostEl" class="terminal-host"></div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.terminal-wrap {
  flex: 1;
  min-height: 0;
  padding: 8px 6px 6px 10px;
  background: #0b1017;
  overflow: hidden;
}

.terminal-host {
  width: 100%;
  height: 100%;
}

.terminal-host :deep(.xterm) {
  height: 100%;
}
</style>
