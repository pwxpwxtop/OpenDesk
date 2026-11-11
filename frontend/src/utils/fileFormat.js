const SIZE_UNITS = ['KB', 'MB', 'GB', 'TB']

const TEXT_EXTENSIONS = ['log', 'txt', 'md', 'conf', 'ini', 'env']
const CODE_EXTENSIONS = ['sh', 'bash', 'py', 'js', 'ts', 'go', 'rs', 'c', 'cpp', 'h', 'java', 'json', 'yml', 'yaml', 'toml', 'xml']
const IMAGE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico']
const ARCHIVE_EXTENSIONS = ['zip', 'tar', 'gz', 'tgz', 'rar', '7z', 'xz', 'bz2']

export function extensionOf(name) {
  const index = name.lastIndexOf('.')
  return index > 0 ? name.slice(index + 1).toLowerCase() : ''
}

export function formatSize(size) {
  if (!Number.isFinite(size)) return '—'
  if (size < 1024) return `${size} B`
  let value = size / 1024
  let index = 0
  while (value >= 1024 && index < SIZE_UNITS.length - 1) {
    value /= 1024
    index += 1
  }
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${SIZE_UNITS[index]}`
}

export function formatTime(seconds) {
  if (!seconds) return ''
  return new Date(seconds * 1000).toLocaleString('zh-CN', { hour12: false })
}

export function iconFor(entry) {
  if (entry.isDir) return 'fa-solid fa-folder'
  if (entry.isLink) return 'fa-solid fa-link'
  const ext = extensionOf(entry.name)
  if (IMAGE_EXTENSIONS.includes(ext)) return 'fa-regular fa-image'
  if (ARCHIVE_EXTENSIONS.includes(ext)) return 'fa-regular fa-file-zipper'
  if (CODE_EXTENSIONS.includes(ext)) return 'fa-regular fa-file-code'
  if (TEXT_EXTENSIONS.includes(ext)) return 'fa-regular fa-file-lines'
  return 'fa-regular fa-file'
}
