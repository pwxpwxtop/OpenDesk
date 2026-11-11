import { computed, reactive, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { SSHService } from '../services/ssh'

function joinPath(base, name) {
  return base === '/' ? `/${name}` : `${base}/${name}`
}

function parentOf(value) {
  const trimmed = value.replace(/\/+$/, '')
  if (!trimmed || trimmed === '/') return '/'
  return trimmed.slice(0, trimmed.lastIndexOf('/')) || '/'
}

function errorText(err) {
  return String(err?.message ?? err)
}

// 文件管理窗口的全部状态与操作，供 FilesApp 及其子组件共享。
export function useFileManager(sessionId) {
  const toast = useToast()

  const cwd = ref('/')
  const entries = ref([])
  const loading = ref(false)
  const error = ref('')

  const selectedPaths = ref([])
  const anchorPath = ref(null)
  const filter = ref('')
  const showHidden = ref(false)
  const sortKey = ref('name')
  const sortDir = ref(1)

  const editingPath = ref(false)
  const pathDraft = ref('')

  // 命名对话框（新建文件夹 / 重命名）的状态，由 FilesApp 里的 NameDialog 渲染。
  const prompt = reactive({
    visible: false,
    title: '新建文件夹',
    label: '',
    value: '',
    placeholder: '',
    confirmText: '确定',
  })
  let promptResolve = null

  function askName(options) {
    Object.assign(prompt, {
      visible: true,
      title: '新建文件夹',
      label: '',
      value: '',
      placeholder: '',
      confirmText: '确定',
      ...options,
    })
    return new Promise((resolve) => {
      promptResolve = resolve
    })
  }

  function submitPrompt(value) {
    prompt.visible = false
    const resolve = promptResolve
    promptResolve = null
    resolve?.(value)
  }

  function cancelPrompt() {
    prompt.visible = false
    const resolve = promptResolve
    promptResolve = null
    resolve?.(null)
  }

  const crumbs = computed(() => {
    const parts = cwd.value.split('/').filter(Boolean)
    const list = [{ label: '/', path: '/' }]
    let acc = ''
    for (const part of parts) {
      acc += `/${part}`
      list.push({ label: part, path: acc })
    }
    return list
  })

  const visibleEntries = computed(() => {
    const term = filter.value.trim().toLowerCase()
    let list = entries.value
    if (!showHidden.value) list = list.filter((entry) => !entry.name.startsWith('.'))
    if (term) list = list.filter((entry) => entry.name.toLowerCase().includes(term))

    const dir = sortDir.value
    return [...list].sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
      if (sortKey.value === 'size') return (a.size - b.size) * dir
      if (sortKey.value === 'modTime') return (a.modTime - b.modTime) * dir
      return a.name.localeCompare(b.name, 'zh-CN') * dir
    })
  })

  const selectedEntries = computed(() =>
    entries.value.filter((entry) => selectedPaths.value.includes(entry.path))
  )

  const primaryEntry = computed(() => {
    const last = selectedPaths.value[selectedPaths.value.length - 1]
    return entries.value.find((entry) => entry.path === last) ?? null
  })

  const selectedSize = computed(() =>
    selectedEntries.value.reduce((total, entry) => total + (entry.isDir ? 0 : entry.size || 0), 0)
  )

  async function load(target) {
    if (!sessionId) return
    loading.value = true
    error.value = ''
    selectedPaths.value = []
    anchorPath.value = null
    try {
      entries.value = await SSHService.ListDir(sessionId, target)
      cwd.value = target
    } catch (err) {
      error.value = errorText(err)
    } finally {
      loading.value = false
    }
  }

  async function init() {
    if (!sessionId) return
    try {
      const home = await SSHService.HomeDir(sessionId)
      await load(home || '/')
    } catch {
      await load('/')
    }
  }

  async function enter(entry) {
    if (entry?.isDir) await load(entry.path)
  }

  async function goUp() {
    await load(parentOf(cwd.value))
  }

  async function refresh() {
    const keep = [...selectedPaths.value]
    await load(cwd.value)
    selectedPaths.value = keep.filter((item) => entries.value.some((entry) => entry.path === item))
  }

  function startEditPath() {
    pathDraft.value = cwd.value
    editingPath.value = true
  }

  async function submitPath(value) {
    const target = String(value ?? pathDraft.value).trim()
    editingPath.value = false
    if (target) await load(target.startsWith('/') ? target : joinPath(cwd.value, target))
  }

  function cancelEditPath() {
    editingPath.value = false
  }

  // ---------- 选择 ----------

  function isSelected(entry) {
    return selectedPaths.value.includes(entry.path)
  }

  function selectRange(fromPath, toPath) {
    const paths = visibleEntries.value.map((entry) => entry.path)
    const from = paths.indexOf(fromPath)
    const to = paths.indexOf(toPath)
    if (from < 0 || to < 0) return false
    const [start, end] = from < to ? [from, to] : [to, from]
    selectedPaths.value = paths.slice(start, end + 1)
    return true
  }

  function selectEntry(entry, event) {
    if (!entry) {
      clearSelection()
      return
    }
    if (event?.shiftKey && anchorPath.value) {
      if (selectRange(anchorPath.value, entry.path)) return
    }
    if (event?.ctrlKey || event?.metaKey) {
      selectedPaths.value = isSelected(entry)
        ? selectedPaths.value.filter((item) => item !== entry.path)
        : [...selectedPaths.value, entry.path]
      anchorPath.value = entry.path
      return
    }
    selectedPaths.value = [entry.path]
    anchorPath.value = entry.path
  }

  function selectAll() {
    selectedPaths.value = visibleEntries.value.map((entry) => entry.path)
    anchorPath.value = selectedPaths.value[selectedPaths.value.length - 1] ?? null
  }

  function clearSelection() {
    selectedPaths.value = []
    anchorPath.value = null
  }

  function moveSelection(delta, extend) {
    const list = visibleEntries.value
    if (!list.length) return
    const currentPath = selectedPaths.value[selectedPaths.value.length - 1]
    const currentIndex = list.findIndex((entry) => entry.path === currentPath)
    const nextIndex = Math.min(Math.max(currentIndex + delta, 0), list.length - 1)
    const next = list[nextIndex]
    if (!next) return
    if (extend && anchorPath.value) {
      selectRange(anchorPath.value, next.path)
      return
    }
    selectedPaths.value = [next.path]
    anchorPath.value = next.path
  }

  // ---------- 文件操作 ----------

  async function makeDir() {
    const name = await askName({
      title: '新建文件夹',
      label: `在 ${cwd.value} 中创建`,
      placeholder: '请输入文件夹名称',
      confirmText: '创建',
    })
    if (!name) return
    try {
      await SSHService.Mkdir(sessionId, joinPath(cwd.value, name))
      await refresh()
    } catch (err) {
      toast.add({ severity: 'error', summary: '创建失败', detail: errorText(err), life: 4000 })
    }
  }

  // 选择本地文件/文件夹后由后端后台上传，完成后通过 ssh:upload:summary 事件刷新目录。
  async function upload() {
    try {
      await SSHService.PickAndUpload(sessionId, cwd.value)
    } catch (err) {
      toast.add({ severity: 'error', summary: '无法选择文件', detail: errorText(err), life: 4000 })
    }
  }

  // 粘贴资源管理器里 Ctrl+C 复制的文件/文件夹到当前目录。
  async function paste() {
    try {
      await SSHService.PasteLocalFiles(sessionId, cwd.value)
    } catch (err) {
      toast.add({ severity: 'warn', summary: '粘贴失败', detail: errorText(err), life: 3600 })
    }
  }

  async function download() {
    const target = primaryEntry.value
    if (!target || target.isDir) return
    try {
      const localPath = await SSHService.Download(sessionId, target.path)
      if (localPath) {
        toast.add({ severity: 'success', summary: '下载完成', detail: localPath, life: 2600 })
      }
    } catch (err) {
      toast.add({ severity: 'error', summary: '下载失败', detail: errorText(err), life: 4000 })
    }
  }

  async function rename() {
    const target = primaryEntry.value
    if (!target) return
    const name = await askName({
      title: '重命名',
      label: target.path,
      value: target.name,
      confirmText: '重命名',
    })
    if (!name || name === target.name) return
    try {
      await SSHService.Rename(sessionId, target.path, joinPath(cwd.value, name))
      await refresh()
    } catch (err) {
      toast.add({ severity: 'error', summary: '重命名失败', detail: errorText(err), life: 4000 })
    }
  }

  async function remove() {
    const targets = [...selectedEntries.value]
    if (!targets.length) return
    const label =
      targets.length === 1
        ? `删除「${targets[0].name}」${targets[0].isDir ? '（含目录内容）' : ''}？`
        : `删除选中的 ${targets.length} 项？`
    if (!window.confirm(label)) return
    try {
      for (const target of targets) {
        await SSHService.Remove(sessionId, target.path, target.isDir)
      }
      await refresh()
    } catch (err) {
      toast.add({ severity: 'error', summary: '删除失败', detail: errorText(err), life: 4000 })
      await refresh()
    }
  }

  function toggleSort(key) {
    if (sortKey.value === key) {
      sortDir.value *= -1
      return
    }
    sortKey.value = key
    sortDir.value = 1
  }

  return {
    // state
    cwd,
    entries,
    loading,
    error,
    selectedPaths,
    filter,
    showHidden,
    sortKey,
    sortDir,
    editingPath,
    pathDraft,
    prompt,
    // computed
    crumbs,
    visibleEntries,
    selectedEntries,
    selectedSize,
    primaryEntry,
    // actions
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
  }
}
