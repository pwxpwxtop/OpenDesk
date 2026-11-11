import { defineStore } from 'pinia'

// Active SSH sessions (runtime only, never persisted: they die with the process).
export const useSessionStore = defineStore('sessions', {
  state: () => ({
    sessions: [],
  }),

  getters: {
    byId: (state) => (sessionId) => state.sessions.find((item) => item.sessionId === sessionId) ?? null,
  },

  actions: {
    add(session) {
      this.remove(session.sessionId)
      this.sessions.push(session)
    },

    remove(sessionId) {
      this.sessions = this.sessions.filter((item) => item.sessionId !== sessionId)
    },
  },
})
