import { defineStore } from 'pinia'
import { APP_META, DESKTOP_ICONS } from '../config/apps'
import { DEFAULT_WALLPAPER_ID, findWallpaper } from '../config/wallpapers'

export { APP_META }

const TASKBAR_HEIGHT = 52

// 桌面图标尺寸与间距，用于计算默认位置和拖拽吸附。
const ICON_WIDTH = 96
const ICON_HEIGHT = 98
const ICON_GAP = 4
const ICON_PADDING = 16
const ICON_STORAGE_KEY = 'opendesk.desktop.icons.v1'
const WALLPAPER_STORAGE_KEY = 'opendesk.desktop.wallpaper.v1'
const FILES_STORAGE_PREFIX = 'opendesk.desktop.files.v1.'

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max)
}

function iconAreaHeight() {
  return Math.max(ICON_HEIGHT * 2, window.innerHeight - TASKBAR_HEIGHT)
}

function iconAreaWidth() {
  return Math.max(ICON_WIDTH * 2, window.innerWidth)
}

// 默认按列自上而下排列，超出一列后向右扩展。
function defaultIconPosition(index) {
  const rows = Math.max(1, Math.floor((iconAreaHeight() - ICON_PADDING * 2) / (ICON_HEIGHT + ICON_GAP)))
  const column = Math.floor(index / rows)
  const row = index % rows
  return {
    x: ICON_PADDING + column * (ICON_WIDTH + ICON_GAP),
    y: ICON_PADDING + row * (ICON_HEIGHT + ICON_GAP),
  }
}

function loadIconPositions() {
  try {
    const parsed = JSON.parse(localStorage.getItem(ICON_STORAGE_KEY) || 'null')
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function buildAppIcons() {
  const saved = loadIconPositions()
  return DESKTOP_ICONS.map((key, index) => {
    const fallback = defaultIconPosition(index)
    const stored = saved[key]
    return {
      key,
      type: 'app',
      app: key,
      x: Number.isFinite(stored?.x) ? stored.x : fallback.x,
      y: Number.isFinite(stored?.y) ? stored.y : fallback.y,
    }
  })
}

function loadFileIcons(sessionId) {
  if (!sessionId) return []
  try {
    const parsed = JSON.parse(localStorage.getItem(FILES_STORAGE_PREFIX + sessionId) || 'null')
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((item) => item && typeof item.path === 'string')
      .map((item) => ({
        key: `file:${item.path}`,
        type: 'file',
        name: item.name || item.path.split('/').pop() || item.path,
        path: item.path,
        isDir: !!item.isDir,
        x: Number(item.x) || ICON_PADDING,
        y: Number(item.y) || ICON_PADDING,
      }))
  } catch {
    return []
  }
}

function loadWallpaper() {
  try {
    const parsed = JSON.parse(localStorage.getItem(WALLPAPER_STORAGE_KEY) || 'null')
    if (parsed && typeof parsed === 'object') {
      return {
        id: typeof parsed.id === 'string' ? parsed.id : DEFAULT_WALLPAPER_ID,
        image: typeof parsed.image === 'string' ? parsed.image : '',
      }
    }
  } catch {
    // 使用默认壁纸。
  }
  return { id: DEFAULT_WALLPAPER_ID, image: '' }
}

// Window manager for one remote desktop window (each session window has its
// own Pinia instance because every native window runs its own webview).
export const useDesktopStore = defineStore('desktop', {
  state: () => ({
    windows: [],
    activeId: null,
    zCounter: 10,
    seq: 0,
    startMenuOpen: false,
    selectedIcon: null,
    sessionId: '',
    appIcons: buildAppIcons(),
    fileIcons: [],
    revealPath: '',
    previewTarget: null,
    wallpaper: loadWallpaper(),
  }),

  getters: {
    hasWindows: (state) => state.windows.length > 0,
    // 应用图标在前，上传到远程的文件/文件夹图标在后。
    icons: (state) => [...state.appIcons, ...state.fileIcons],
    wallpaperStyle: (state) => {
      if (state.wallpaper.image) {
        return { background: `#0a1020 url("${state.wallpaper.image}") center / cover no-repeat` }
      }
      return { background: findWallpaper(state.wallpaper.id).background }
    },
  },

  actions: {
    openWindow(app) {
      const meta = APP_META[app]
      if (!meta) return null

      const existing = this.windows.find((win) => win.app === app)
      if (existing) {
        this.focus(existing.id)
        return existing
      }

      const areaHeight = window.innerHeight - TASKBAR_HEIGHT
      const width = Math.min(meta.width, window.innerWidth - 60)
      const height = Math.min(meta.height, areaHeight - 30)
      const offset = this.windows.length * 26

      const win = {
        id: `${app}-${(this.seq += 1)}`,
        app,
        title: meta.title,
        icon: meta.icon,
        x: Math.max(20, Math.round((window.innerWidth - width) / 2) + offset),
        y: Math.max(16, Math.round((areaHeight - height) / 2) + offset - 10),
        width,
        height,
        z: (this.zCounter += 1),
        minimized: false,
        maximized: false,
      }

      this.windows.push(win)
      this.activeId = win.id
      this.startMenuOpen = false
      this.selectedIcon = null
      return win
    },

    focus(id) {
      const win = this.windows.find((item) => item.id === id)
      if (!win) return
      win.minimized = false
      win.z = (this.zCounter += 1)
      this.activeId = id
      this.startMenuOpen = false
    },

    minimize(id) {
      const win = this.windows.find((item) => item.id === id)
      if (!win) return
      win.minimized = true
      if (this.activeId === id) this.activeId = null
    },

    close(id) {
      const win = this.windows.find((item) => item.id === id)
      this.windows = this.windows.filter((item) => item.id !== id)
      if (this.activeId === id) {
        const top = this.windows.filter((item) => !item.minimized).sort((a, b) => b.z - a.z)[0]
        this.activeId = top ? top.id : null
      }
      return win
    },

    toggleMaximize(id) {
      const win = this.windows.find((item) => item.id === id)
      if (!win) return
      if (!win.maximized) {
        win.restore = { x: win.x, y: win.y, width: win.width, height: win.height }
        win.maximized = true
      } else {
        win.maximized = false
        if (win.restore) Object.assign(win, win.restore)
      }
      this.focus(id)
    },

    setBounds(id, patch) {
      const win = this.windows.find((item) => item.id === id)
      if (!win) return
      win.width = clamp(patch.width ?? win.width, 420, window.innerWidth - 8)
      win.height = clamp(patch.height ?? win.height, 260, window.innerHeight - TASKBAR_HEIGHT - 8)
      win.x = clamp(patch.x ?? win.x, -win.width + 160, window.innerWidth - 160)
      win.y = clamp(patch.y ?? win.y, 0, window.innerHeight - TASKBAR_HEIGHT - 60)
    },

    closeAll() {
      this.windows = []
      this.activeId = null
    },

    toggleStartMenu() {
      this.startMenuOpen = !this.startMenuOpen
    },

    closeStartMenu() {
      this.startMenuOpen = false
    },

    // ---------- 桌面图标 ----------

    selectIcon(key) {
      this.selectedIcon = key
    },

    clearIconSelection() {
      this.selectedIcon = null
    },

    // 拖拽过程中调用，只更新位置，不做持久化。
    moveIcon(key, x, y) {
      const icon = this.icons.find((item) => item.key === key)
      if (!icon) return
      icon.x = clamp(x, 4, iconAreaWidth() - ICON_WIDTH - 4)
      icon.y = clamp(y, 4, iconAreaHeight() - ICON_HEIGHT)
    },

    persistIcons() {
      const layout = {}
      for (const icon of this.icons) {
        layout[icon.key] = { x: Math.round(icon.x), y: Math.round(icon.y) }
      }
      try {
        localStorage.setItem(ICON_STORAGE_KEY, JSON.stringify(layout))
      } catch {
        // 隐私模式下 localStorage 不可用，忽略即可。
      }
    },

    resetIcons() {
      this.appIcons = DESKTOP_ICONS.map((key, index) => ({
        key,
        type: 'app',
        app: key,
        ...defaultIconPosition(index),
      }))
      this.fileIcons = this.fileIcons.map((icon, index) => ({
        ...icon,
        ...defaultIconPosition(DESKTOP_ICONS.length + index),
      }))
      this.selectedIcon = null
      this.persistIcons()
      this.persistFiles()
    },

    // ---------- 远程文件图标 ----------

    attachSession(sessionId) {
      this.sessionId = sessionId
      this.fileIcons = loadFileIcons(sessionId)
    },

    // 上传完成后把顶层文件/文件夹显示到桌面上（已存在则跳过）。
    addFileIcons(items) {
      if (!Array.isArray(items) || items.length === 0) return
      for (const item of items) {
        if (!item?.path) continue
        const key = `file:${item.path}`
        if (this.fileIcons.some((icon) => icon.key === key)) continue
        const position = defaultIconPosition(this.appIcons.length + this.fileIcons.length)
        this.fileIcons.push({
          key,
          type: 'file',
          name: item.name || item.path.split('/').pop() || item.path,
          path: item.path,
          isDir: !!item.isDir,
          ...position,
        })
      }
      this.persistFiles()
    },

    removeFileIcon(key) {
      this.fileIcons = this.fileIcons.filter((icon) => icon.key !== key)
      if (this.selectedIcon === key) this.selectedIcon = null
      this.persistFiles()
    },

    // 在文件管理窗口里定位到某个远程路径。
    revealInFiles(target) {
      this.revealPath = target
      this.openWindow('files')
    },

    // 打开预览窗口（视频播放 / 文档查看），同一时刻只保留一个预览窗口。
    openPreview(file) {
      this.previewTarget = { ...file }
      const existing = this.windows.find((win) => win.app === 'preview')
      if (existing) {
        existing.title = file.name
        this.focus(existing.id)
        return existing
      }
      const win = this.openWindow('preview')
      if (win) win.title = file.name
      return win
    },

    persistFiles() {
      if (!this.sessionId) return
      const payload = this.fileIcons.map((icon) => ({
        name: icon.name,
        path: icon.path,
        isDir: icon.isDir,
        x: Math.round(icon.x),
        y: Math.round(icon.y),
      }))
      try {
        localStorage.setItem(FILES_STORAGE_PREFIX + this.sessionId, JSON.stringify(payload))
      } catch {
        // 忽略存储失败。
      }
    },

    // ---------- 壁纸 ----------

    applyWallpaper(id) {
      this.wallpaper = { id, image: '' }
      this.persistWallpaper()
    },

    // 返回 false 表示本地存储空间不足（图片太大），壁纸仅在本次窗口有效。
    applyWallpaperImage(dataUrl) {
      this.wallpaper = { id: this.wallpaper.id, image: dataUrl }
      return this.persistWallpaper()
    },

    resetWallpaper() {
      this.wallpaper = { id: DEFAULT_WALLPAPER_ID, image: '' }
      this.persistWallpaper()
    },

    persistWallpaper() {
      try {
        localStorage.setItem(WALLPAPER_STORAGE_KEY, JSON.stringify(this.wallpaper))
        return true
      } catch {
        return false
      }
    },
  },
})
