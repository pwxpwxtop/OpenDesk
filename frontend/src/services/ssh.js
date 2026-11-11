import { Events } from '@wailsio/runtime'
import { SSHService } from '../../bindings/changeme'

const MAX_BUFFER = 512 * 1024

const buffers = new Map()
const listeners = new Map()
const closedHandlers = new Set()
const uploadHandlers = new Set()

let initialized = false

function decodeBase64(value) {
  const binary = atob(value)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

// initSSHBus subscribes to backend SSH events once per app lifetime.
// Output is buffered per session until a terminal attaches, so the login
// banner is never lost when a window opens slightly late.
export function initSSHBus() {
  if (initialized) return
  initialized = true

  Events.On('ssh:output', (event) => {
    const { sessionId, data } = event.data ?? {}
    if (!sessionId || !data) return

    const bytes = decodeBase64(data)
    const attached = listeners.get(sessionId)
    if (attached && attached.size > 0) {
      attached.forEach((handler) => handler(bytes))
      return
    }

    let buffer = buffers.get(sessionId)
    if (!buffer) {
      buffer = { chunks: [], size: 0 }
      buffers.set(sessionId, buffer)
    }
    buffer.chunks.push(bytes)
    buffer.size += bytes.length
    while (buffer.size > MAX_BUFFER && buffer.chunks.length > 0) {
      buffer.size -= buffer.chunks.shift().length
    }
  })

  Events.On('ssh:closed', (event) => {
    const { sessionId, reason } = event.data ?? {}
    buffers.delete(sessionId)
    closedHandlers.forEach((handler) => handler(sessionId, reason))
  })

  Events.On('ssh:upload', (event) => {
    const payload = event.data ?? {}
    uploadHandlers.forEach((handler) => handler('file', payload))
  })

  Events.On('ssh:upload:summary', (event) => {
    const payload = event.data ?? {}
    uploadHandlers.forEach((handler) => handler('summary', payload))
  })
}

// attachOutput replays buffered output and then streams live output.
export function attachOutput(sessionId, onData) {
  const pending = buffers.get(sessionId)
  if (pending) {
    buffers.delete(sessionId)
    pending.chunks.forEach((chunk) => onData(chunk))
  }

  let handlers = listeners.get(sessionId)
  if (!handlers) {
    handlers = new Set()
    listeners.set(sessionId, handlers)
  }
  handlers.add(onData)

  return () => {
    handlers.delete(onData)
    if (handlers.size === 0) listeners.delete(sessionId)
  }
}

export function onSessionClosed(handler) {
  closedHandlers.add(handler)
  return () => closedHandlers.delete(handler)
}

// onUploadResult receives ("file", UploadProgress) and ("summary", UploadSummary).
export function onUploadResult(handler) {
  uploadHandlers.add(handler)
  return () => uploadHandlers.delete(handler)
}

export function connect(profile) {
  return SSHService.Connect({
    host: profile.host,
    port: Number(profile.port) || 22,
    user: profile.user,
    password: profile.password || '',
    privateKeyPath: profile.privateKeyPath || '',
    passphrase: profile.passphrase || '',
  })
}

export function disconnect(sessionId) {
  return SSHService.Disconnect(sessionId)
}

export { SSHService }
