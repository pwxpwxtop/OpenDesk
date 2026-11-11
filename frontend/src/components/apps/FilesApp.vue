<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import ContextMenu from '../desktop/ContextMenu.vue'
import FileToolbar from './files/FileToolbar.vue'
import FileBreadcrumb from './files/FileBreadcrumb.vue'
import FileTable from './files/FileTable.vue'
import FileShortcutHelp from './files/FileShortcutHelp.vue'
import UploadProgress from './files/UploadProgress.vue'
import NameDialog from './files/NameDialog.vue'
import { useFileManager } from '../../composables/useFileManager'
import { useDesktopStore } from '../../stores/desktop'
import { onUploadResult } from '../../services/ssh'
import { formatSize } from '../../utils/fileFormat'
import { isPreviewable } from '../../utils/preview'

const props = defineProps({
  sessionId: { type: String, default: '' },
  host: { type: String, default: '' },
})

const root = ref(null)
const helpVisible = ref(false)
const menu = reactive({ visible: false, x: 0, y: 0, items: [], entry: null })
const desktop = useDesktopStore()
let detachUpload = null

const {
  cwd,
  loading,
  error,
  filter,
  showHidden,
  sortKey,
  sortDir,
  editingPath,
  pathDraft,
  prompt,
  crumbs,
  visibleEntries,
  selectedPaths,
  selectedSize,
  primaryEntry,
  init,
  load,
  enter,
  goUp,
  refresh,
  startEditPath,
  submitPath,
  cancelEditPath,
  isSelected,
  selectEntry,
  selectAll,
  clearSelection,
  moveSelection,
  makeDir,
  upload,
  paste,
  download,
  rename,
  remove,
  toggleSort,
  submitPrompt,
  cancelPrompt,
} = useFileManager(props.sessionId)

const selectedCount = computed(() => selectedPaths.value.length)
const canDownload = computed(
  () => selectedPaths.value.length === 1 && primaryEntry.value && !primaryEntry.value.isDir
)
const canRename = computed(() => selectedPaths.value.length === 1 && !!primaryEntry.value)

// 双击/回车：目录进入，可预览的文件开预览窗口（视频播放、文档查看），其余走下载。
function openEntry(entry) {
  if (!entry) return
  if (entry.isDir) {
    enter(entry)
    return
  }
  if (!isSelected(entry)) selectEntry(entry)
  if (isPreviewable(entry.name)) {
    desktop.openPreview({ name: entry.name, path: entry.path, size: entry.size })
    return
  }
  download()
}

function focusRoot() {
  nextTick(() => root.value?.focus({ preventScroll: true }))
}

function closeMenu() {
  menu.visible = false
}

function rowMenu(entry) {
  const items = []
  if (entry.isDir) {
    items.push({ label: '打开', icon: 'fa-regular fa-folder-open', action: 'open' })
  } else if (isPreviewable(entry.name)) {
    items.push({ label: '预览', icon: 'fa-regular fa-eye', action: 'preview' })
  }
  if (!entry.isDir) {
    items.push({ label: '下载到本地', icon: 'fa-solid fa-download', action: 'download' })
  }
  items.push(
    { separator: true },
    { label: '重命名', icon: 'fa-solid fa-pen', action: 'rename' },
    { label: '删除', icon: 'fa-solid fa-trash', action: 'remove', danger: true },
    { separator: true },
    { label: '刷新', icon: 'fa-solid fa-rotate', action: 'refresh' },
    { label: '新建文件夹', icon: 'fa-solid fa-folder-plus', action: 'mkdir' },
    { label: '上传到当前目录', icon: 'fa-solid fa-upload', action: 'upload' }
  )
  return items
}

function blankMenu() {
  const items = [
    { label: '刷新', icon: 'fa-solid fa-rotate', action: 'refresh' },
    { label: '新建文件夹', icon: 'fa-solid fa-folder-plus', action: 'mkdir' },
    { label: '上传到当前目录', icon: 'fa-solid fa-upload', action: 'upload' },
  ]
  if (selectedCount.value > 0) {
    items.push(
      { separator: true },
      { label: '全选', icon: 'fa-solid fa-check-double', action: 'selectAll' },
      { label: '清除选择', icon: 'fa-solid fa-ban', action: 'clear' }
    )
  }
  return items
}

function openMenu(entry, event) {
  if (entry) {
    if (!isSelected(entry)) selectEntry(entry)
  } else {
    clearSelection()
  }
  menu.entry = entry
  menu.items = entry ? rowMenu(entry) : blankMenu()

  const rect = root.value?.getBoundingClientRect() ?? { left: 0, top: 0 }
  const width = root.value?.clientWidth ?? 860
  const height = root.value?.clientHeight ?? 560
  menu.x = Math.max(4, Math.min(event.clientX - rect.left, width - 210))
  menu.y = Math.max(4, Math.min(event.clientY - rect.top, height - 260))
  menu.visible = true
  helpVisible.value = false
  focusRoot()
}

function onMenuSelect(action) {
  const entry = menu.entry
  menu.visible = false
  switch (action) {
    case 'open':
      enter(entry)
      break
    case 'preview':
      openEntry(entry)
      break
    case 'download':
      download()
      break
    case 'rename':
      rename()
      break
    case 'remove':
      remove()
      break
    case 'refresh':
      refresh()
      break
    case 'mkdir':
      makeDir()
      break
    case 'upload':
      upload()
      break
    case 'selectAll':
      selectAll()
      break
    case 'clear':
      clearSelection()
      break
  }
  focusRoot()
}

function onRootClick(event) {
  closeMenu()
  if (event.target instanceof HTMLElement && event.target.closest('input, textarea')) return
  focusRoot()
}

function onKeydown(event) {
  if (!props.sessionId) return
  if (event.target instanceof HTMLElement && event.target.closest('input, textarea')) return

  const key = event.key
  const mod = event.ctrlKey || event.metaKey

  if (key === 'Escape') {
    event.preventDefault()
    if (helpVisible.value) helpVisible.value = false
    else if (menu.visible) menu.visible = false
    else if (filter.value) filter.value = ''
    else clearSelection()
    return
  }

  if (key === 'F2') {
    event.preventDefault()
    rename()
    return
  }
  if (key === 'Delete') {
    event.preventDefault()
    remove()
    return
  }
  if (key === 'F5') {
    event.preventDefault()
    refresh()
    return
  }
  if (key === 'Backspace' || (event.altKey && key === 'ArrowLeft')) {
    event.preventDefault()
    goUp()
    return
  }
  if (key === 'Enter') {
    event.preventDefault()
    openEntry(primaryEntry.value)
    return
  }
  if (key === 'ArrowDown' || key === 'ArrowUp') {
    event.preventDefault()
    moveSelection(key === 'ArrowDown' ? 1 : -1, event.shiftKey)
    return
  }

  if (!mod) return

  switch (key.toLowerCase()) {
    case 'a':
      event.preventDefault()
      selectAll()
      break
    case 'n':
      event.preventDefault()
      makeDir()
      break
    case 'u':
      event.preventDefault()
      upload()
      break
    case 'v':
      event.preventDefault()
      paste()
      break
    case 'r':
      event.preventDefault()
      refresh()
      break
    case 'l':
      event.preventDefault()
      startEditPath()
      break
  }
}

onMounted(async () => {
  if (!props.sessionId) return
  detachUpload = onUploadResult((kind, payload) => {
    if (kind !== 'summary' || payload?.sessionId !== props.sessionId) return
    if (payload.remoteDir === cwd.value) refresh()
  })
  await init()
  focusRoot()
})

// 双击桌面上的文件/文件夹图标时，定位到对应远程目录。
watch(
  () => desktop.revealPath,
  async (target) => {
    if (!target) return
    desktop.revealPath = ''
    if (target !== cwd.value) await load(target)
    focusRoot()
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  detachUpload?.()
})
</script>

<template>
  <div v-if="!sessionId" class="app-empty">
    <span class="glyph"><i class="fa-regular fa-folder-open"></i></span>
    <div>这个窗口没有关联的 SSH 会话。</div>
  </div>

  <div
    v-else
    ref="root"
    class="app-root file-app"
    tabindex="-1"
    @keydown="onKeydown"
    @click="onRootClick"
  >
    <FileToolbar
      :busy="loading"
      :selected-count="selectedCount"
      :can-download="canDownload"
      :can-rename="canRename"
      :filter="filter"
      :show-hidden="showHidden"
      @up="goUp"
      @refresh="refresh"
      @mkdir="makeDir"
      @upload="upload"
      @download="download"
      @rename="rename"
      @remove="remove"
      @show-shortcuts="helpVisible = !helpVisible"
      @update:filter="filter = $event"
      @update:show-hidden="showHidden = $event"
    />

    <FileBreadcrumb
      :crumbs="crumbs"
      :editing="editingPath"
      :draft="pathDraft"
      @navigate="load"
      @start-edit="startEditPath"
      @submit="submitPath"
      @cancel="cancelEditPath"
      @update:draft="pathDraft = $event"
    />

    <FileTable
      :entries="visibleEntries"
      :selected-paths="selectedPaths"
      :loading="loading"
      :sort-key="sortKey"
      :sort-dir="sortDir"
      data-file-drop-target="files"
      :data-drop-dir="cwd"
      @select="selectEntry"
      @open="openEntry"
      @sort="toggleSort"
      @context="openMenu"
    />

    <p v-if="error" class="file-error">{{ error }}</p>

    <UploadProgress :session-id="sessionId" />

    <div class="app-footer">
      <span>
        {{ visibleEntries.length }} 项
        <template v-if="selectedCount > 0">
          · 已选 {{ selectedCount }} 项
          <template v-if="selectedSize > 0">（{{ formatSize(selectedSize) }}）</template>
        </template>
      </span>
      <span class="mono">{{ host }}</span>
    </div>

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menu.items"
      @select="onMenuSelect"
    />

    <FileShortcutHelp :visible="helpVisible" @close="helpVisible = false" />

    <NameDialog
      :visible="prompt.visible"
      v-model="prompt.value"
      :title="prompt.title"
      :label="prompt.label"
      :placeholder="prompt.placeholder"
      :confirm-text="prompt.confirmText"
      @confirm="submitPrompt"
      @cancel="cancelPrompt"
    />
  </div>
</template>
