import { defineStore } from 'pinia'

// 上传进度：由 ssh:upload / ssh:upload:summary 事件驱动，文件管理窗口据此渲染进度条。
export const useTransferStore = defineStore('transfers', {
  state: () => ({
    sessionId: '',
    remoteDir: '',
    active: false,
    done: 0,
    failed: 0,
    current: null,
    lastError: '',
    finishedAt: 0,
  }),

  getters: {
    percent: (state) => {
      if (!state.current || !state.current.size) return 0
      return Math.min(100, Math.round((state.current.transferred / state.current.size) * 100))
    },
  },

  actions: {
    handle(kind, payload) {
      if (!payload) return
      if (kind === 'file') {
        if (!payload.sessionId) return
        this.sessionId = payload.sessionId
        this.remoteDir = payload.remoteDir || this.remoteDir
        this.active = true
        if (payload.error) {
          this.failed += 1
          this.lastError = payload.error
          return
        }
        if (payload.done) {
          this.done += 1
          this.current = null
          return
        }
        this.current = {
          name: payload.name,
          transferred: payload.transferred ?? 0,
          size: payload.size ?? 0,
        }
        return
      }

      if (kind === 'summary') {
        if (!payload.sessionId || payload.sessionId !== this.sessionId) return
        this.current = null
        this.active = false
        this.failed = payload.failed ?? this.failed
        this.finishedAt = Date.now()
      }
    },

    reset() {
      this.sessionId = ''
      this.remoteDir = ''
      this.active = false
      this.done = 0
      this.failed = 0
      this.current = null
      this.lastError = ''
      this.finishedAt = 0
    },
  },
})
