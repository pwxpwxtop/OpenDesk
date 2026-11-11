<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import WindowFrame from '../components/desktop/WindowFrame.vue'
import DesktopIcon from '../components/desktop/DesktopIcon.vue'
import TaskBar from '../components/desktop/TaskBar.vue'
import ContextMenu from '../components/desktop/ContextMenu.vue'
import TerminalApp from '../components/apps/TerminalApp.vue'
import FilesApp from '../components/apps/FilesApp.vue'
import MonitorApp from '../components/apps/MonitorApp.vue'
import WallpaperApp from '../components/apps/WallpaperApp.vue'
import PreviewApp from '../components/apps/PreviewApp.vue'
import { useDesktopStore } from '../stores/desktop'
import { APP_META } from '../config/apps'
import { isPreviewable } from '../utils/preview'
import { disconnect, onSessionClosed, onUploadResult, SSHService } from '../services/ssh'

const route = useRoute()
const sessionId = String(route.params.sessionId ?? '')
const host = String(route.query.host ?? '')
const desktop = useDesktopStore()
const toast = useToast()

const connected = ref(true)
const menu = reactive({ visible: false, x: 0, y: 0, items: [], icon: null })
let detachClosed = null
let detachUpload = null

const appComponents = {
  terminal: TerminalApp,
  files: FilesApp,
  monitor: MonitorApp,
  wallpaper: WallpaperApp,
  preview: PreviewApp,
}

const menuItems = [
  { label: '打开控制台', icon: 'fa-solid fa-terminal', action: 'terminal' },
  { label: '文件管理', icon: 'fa-regular fa-folder-open', action: 'files' },
  { label: '系统监控', icon: 'fa-solid fa-chart-line', action: 'monitor' },
  { label: '更换壁纸', icon: 'fa-regular fa-image', action: 'wallpaper' },
  { label: '上传本地文件…', icon: 'fa-solid fa-upload', action: 'upload' },
  { label: '粘贴本地文件', icon: 'fa-regular fa-clipboard', action: 'paste' },
  { separator: true },
  { label: '整理桌面图标', icon: 'fa-solid fa-table-cells-large', action: 'tidy' },
  { label: '关闭全部窗口', icon: 'fa-regular fa-rectangle-xmark', action: 'closeAll' },
  { label: '断开连接', icon: 'fa-solid fa-power-off', action: 'disconnect', danger: true },
]

// 桌面图标的外观：应用图标取 APP_META，远程文件图标用文件夹/文件图标。
function iconMeta(icon) {
  if (icon.type === 'file') {
    return icon.isDir
      ? { label: icon.name, icon: 'fa-solid fa-folder', from: '#fbbf24', to: '#d97706' }
      : { label: icon.name, icon: 'fa-regular fa-file-lines', from: '#60a5fa', to: '#2563eb' }
  }
  const meta = APP_META[icon.app]
  return { label: meta.title, icon: meta.icon, from: meta.from, to: meta.to }
}

function parentPath(value) {
  const trimmed = String(value || '').replace(/\/+$/, '')
  if (!trimmed || trimmed === '/') return '/'
  return trimmed.slice(0, trimmed.lastIndexOf('/')) || '/'
}

function openIcon(icon) {
  if (icon.type === 'app') {
    desktop.openWindow(icon.app)
    return
  }
  // 可预览的文件直接打开预览窗口，其它则定位到所在目录。
  if (!icon.isDir && isPreviewable(icon.name)) {
    desktop.openPreview({ name: icon.name, path: icon.path, size: 0 })
    return
  }
  desktop.revealInFiles(icon.isDir ? icon.path : parentPath(icon.path))
}

function onIconMenu(icon, event) {
  desktop.selectIcon(icon.key)
  const items = [
    {
      label: icon.isDir ? '打开文件夹' : '打开所在目录',
      icon: 'fa-regular fa-folder-open',
      action: 'reveal',
    },
  ]
  if (!icon.isDir) {
    items.push({ label: '下载到本地', icon: 'fa-solid fa-download', action: 'download' })
  }
  items.push({ separator: true }, { label: '从桌面移除', icon: 'fa-solid fa-xmark', action: 'remove', danger: true })

  menu.icon = icon
  menu.items = items
  menu.x = Math.min(event.clientX, window.innerWidth - 210)
  menu.y = Math.min(event.clientY, window.innerHeight - 260)
  menu.visible = true
  desktop.closeStartMenu()
}

async function downloadIcon(icon) {
  try {
    await SSHService.Download(sessionId, icon.path)
  } catch (error) {
    toast.add({ severity: 'error', summary: '下载失败', detail: String(error?.message ?? error), life: 4200 })
  }
}

// 拖拽结束后吸附到桌面网格：96px 图标 + 4px 间距。
const GRID_STEP_X = 100
const GRID_STEP_Y = 102
const GRID_ORIGIN = 16

function onIconDrop(key, position) {
  desktop.moveIcon(
    key,
    GRID_ORIGIN + Math.round((position.x - GRID_ORIGIN) / GRID_STEP_X) * GRID_STEP_X,
    GRID_ORIGIN + Math.round((position.y - GRID_ORIGIN) / GRID_STEP_Y) * GRID_STEP_Y
  )
  desktop.persistIcons()
  desktop.persistFiles()
}

onMounted(() => {
  desktop.attachSession(sessionId)
  detachClosed = onSessionClosed((closedId) => {
    if (closedId === sessionId) connected.value = false
  })
  detachUpload = onUploadResult((kind, payload) => {
    if (kind !== 'summary' || payload?.sessionId !== sessionId) return
    // 复制到远程的文件/文件夹直接显示在桌面上。
    desktop.addFileIcons(payload.items)
    if (payload.uploaded > 0) {
      toast.add({
        severity: 'success',
        summary: `已上传 ${payload.uploaded} 个文件`,
        detail: payload.remoteDir,
        life: 3000,
      })
    }
    if (payload.failed > 0) {
      toast.add({
        severity: 'warn',
        summary: `${payload.failed} 个文件上传失败`,
        detail: payload.remoteDir,
        life: 4200,
      })
    }
  })
  window.addEventListener('click', closeMenus)
  window.addEventListener('keydown', onWindowKeydown)
})

onBeforeUnmount(() => {
  detachClosed?.()
  detachUpload?.()
  window.removeEventListener('click', closeMenus)
  window.removeEventListener('keydown', onWindowKeydown)
})

// 桌面空白处按 Ctrl+V：粘贴资源管理器里复制的文件到远程 ~/Desktop。
// 焦点在应用窗口内时交给窗口自己处理（文件管理有同款快捷键）。
function onWindowKeydown(event) {
  const mod = event.ctrlKey || event.metaKey
  if (!mod || event.key.toLowerCase() !== 'v') return
  const target = event.target
  if (target instanceof HTMLElement) {
    if (target.closest('.window')) return
    if (target.closest('input, textarea, [contenteditable="true"]')) return
  }
  event.preventDefault()
  pasteLocalFiles()
}

function closeMenus() {
  menu.visible = false
  desktop.closeStartMenu()
}

function onDesktopClick(event) {
  if (event.target !== event.currentTarget) return
  menu.visible = false
  desktop.closeStartMenu()
  desktop.clearIconSelection()
}

function onContextMenu(event) {
  if (event.target !== event.currentTarget) return
  event.preventDefault()
  menu.icon = null
  menu.items = menuItems
  menu.x = Math.min(event.clientX, window.innerWidth - 210)
  menu.y = Math.min(event.clientY, window.innerHeight - 300)
  menu.visible = true
  desktop.closeStartMenu()
  desktop.clearIconSelection()
}

function onMenuSelect(action) {
  const icon = menu.icon
  menu.visible = false
  if (icon) {
    menu.icon = null
    if (action === 'reveal') openIcon(icon)
    else if (action === 'download') downloadIcon(icon)
    else if (action === 'remove') desktop.removeFileIcon(icon.key)
    return
  }
  if (['terminal', 'files', 'monitor', 'wallpaper'].includes(action)) {
    desktop.openWindow(action)
    return
  }
  if (action === 'tidy') {
    desktop.resetIcons()
    return
  }
  if (action === 'upload') {
    pickLocalFiles()
    return
  }
  if (action === 'paste') {
    pasteLocalFiles()
    return
  }
  if (action === 'closeAll') {
    desktop.closeAll()
    return
  }
  if (action === 'disconnect') disconnectSession()
}

// 通过系统文件选择框上传到远程桌面目录 ~/Desktop（也可以直接从资源管理器拖进桌面窗口）。
async function pickLocalFiles() {
  try {
    await SSHService.PickAndUpload(sessionId, 'desktop')
  } catch (error) {
    toast.add({ severity: 'error', summary: '无法选择文件', detail: String(error?.message ?? error), life: 4200 })
  }
}

// 把资源管理器里 Ctrl+C 复制的文件粘贴到远程 ~/Desktop。
async function pasteLocalFiles() {
  try {
    await SSHService.PasteLocalFiles(sessionId, 'desktop')
  } catch (error) {
    toast.add({ severity: 'warn', summary: '粘贴失败', detail: String(error?.message ?? error), life: 3600 })
  }
}

async function disconnectSession() {
  await disconnect(sessionId).catch(() => {})
  connected.value = false
}
</script>

<template>
  <div
    class="session-desktop"
    :style="desktop.wallpaperStyle"
    data-file-drop-target="desktop"
    data-drop-dir="desktop"
    @click="onDesktopClick"
    @contextmenu="onContextMenu"
  >
    <div class="desktop-icons">
      <DesktopIcon
        v-for="icon in desktop.icons"
        :key="icon.key"
        :label="iconMeta(icon).label"
        :icon="iconMeta(icon).icon"
        :from="iconMeta(icon).from"
        :to="iconMeta(icon).to"
        :x="icon.x"
        :y="icon.y"
        :selected="desktop.selectedIcon === icon.key"
        @select="desktop.selectIcon(icon.key)"
        @open="openIcon(icon)"
        @menu="onIconMenu(icon, $event)"
        @move="desktop.moveIcon(icon.key, $event.x, $event.y)"
        @moveend="onIconDrop(icon.key, $event)"
      />
    </div>

    <div class="windows-layer">
      <WindowFrame v-for="win in desktop.windows" :key="win.id" :win="win">
        <component
          :is="appComponents[win.app]"
          :session-id="sessionId"
          :host="host"
          @open-app="desktop.openWindow($event)"
        />
      </WindowFrame>
    </div>

    <div v-if="!connected" class="disconnect-banner">
      <i class="fa-solid fa-plug-circle-xmark"></i>
      与 {{ host }} 的连接已断开
    </div>

    <TaskBar :host="host" :connected="connected" @disconnect="disconnectSession" />

    <ContextMenu :visible="menu.visible" :x="menu.x" :y="menu.y" :items="menu.items" @select="onMenuSelect" />
  </div>
</template>
