// 文件预览能力判断：视频/音频/图片/PDF 走流式接口，文本类受 1MB 大小限制。
const VIDEO_EXTENSIONS = ['mp4', 'webm', 'ogv', 'mov', 'm4v', 'mkv', 'avi']
const AUDIO_EXTENSIONS = ['mp3', 'wav', 'ogg', 'flac', 'm4a', 'aac']
const IMAGE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico', 'avif']
const PDF_EXTENSIONS = ['pdf']
const TEXT_EXTENSIONS = [
  'txt', 'md', 'log', 'json', 'xml', 'yml', 'yaml', 'toml', 'ini', 'conf', 'env', 'csv',
  'sh', 'bash', 'zsh', 'py', 'js', 'ts', 'go', 'rs', 'c', 'cpp', 'h', 'hpp', 'java',
  'html', 'css', 'scss', 'sql', 'service', 'properties', 'dockerfile', 'gitignore',
]

export const PREVIEW_MAX_BYTES = 1024 * 1024

function extensionOf(name) {
  const index = String(name).lastIndexOf('.')
  return index > 0 ? String(name).slice(index + 1).toLowerCase() : ''
}

// 返回 video / audio / image / pdf / text / ''（不支持预览）。
export function previewKind(name) {
  const ext = extensionOf(name)
  if (!ext) return ''
  if (VIDEO_EXTENSIONS.includes(ext)) return 'video'
  if (AUDIO_EXTENSIONS.includes(ext)) return 'audio'
  if (IMAGE_EXTENSIONS.includes(ext)) return 'image'
  if (PDF_EXTENSIONS.includes(ext)) return 'pdf'
  if (TEXT_EXTENSIONS.includes(ext)) return 'text'
  return ''
}

export function isPreviewable(name) {
  return previewKind(name) !== ''
}

// 预览地址：由 Go 端 /preview 路由按 HTTP Range 直接从远程读取。
export function previewUrl(sessionId, remotePath) {
  const query = new URLSearchParams({ session: sessionId, path: remotePath })
  return `/preview?${query.toString()}`
}

export function isTooLargeForText(size) {
  return Number(size) > PREVIEW_MAX_BYTES
}

export function formatSize(size) {
  const value = Number(size)
  if (!Number.isFinite(value)) return '—'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(2)} MB`
}
