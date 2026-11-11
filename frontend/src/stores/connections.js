import { defineStore } from 'pinia'

const PROFILES_KEY = 'opendesk.connections.v1'

export const ACCENT_COLORS = ['#4f8cff', '#22d3ee', '#a855f7', '#f59e0b', '#10b981', '#ef4444']

function loadProfiles() {
  try {
    const list = JSON.parse(localStorage.getItem(PROFILES_KEY) ?? '[]')
    return Array.isArray(list) ? list : []
  } catch (error) {
    return []
  }
}

function emptyProfile() {
  return {
    id: '',
    name: '',
    host: '',
    port: 22,
    user: 'root',
    password: '',
    privateKeyPath: '',
    passphrase: '',
    color: ACCENT_COLORS[0],
  }
}

export const useConnectionStore = defineStore('connections', {
  state: () => ({
    profiles: loadProfiles(),
  }),

  getters: {
    byId: (state) => (id) => state.profiles.find((profile) => profile.id === id) ?? null,
  },

  actions: {
    persist() {
      localStorage.setItem(PROFILES_KEY, JSON.stringify(this.profiles))
    },

    createProfile() {
      return { ...emptyProfile(), id: `conn-${Date.now().toString(36)}` }
    },

    saveProfile(profile) {
      const record = {
        ...profile,
        port: Number(profile.port) || 22,
        name: profile.name?.trim() || `${profile.user}@${profile.host}`,
      }
      const index = this.profiles.findIndex((item) => item.id === record.id)
      if (index >= 0) {
        this.profiles[index] = record
      } else {
        this.profiles.push(record)
      }
      this.persist()
      return record
    },

    removeProfile(id) {
      this.profiles = this.profiles.filter((profile) => profile.id !== id)
      this.persist()
    },
  },
})
