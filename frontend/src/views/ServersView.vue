<script setup>
import { computed, ref } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import { useToast } from 'primevue/usetoast'
import { useConnectionStore } from '../stores/connections'
import { useSessionStore } from '../stores/sessions'
import { connect, disconnect, SSHService } from '../services/ssh'

const conn = useConnectionStore()
const sessions = useSessionStore()
const toast = useToast()

const section = ref('servers')
const keyword = ref('')
const dialogVisible = ref(false)
const form = ref(conn.createProfile())
const connectingId = ref(null)

const isEditing = computed(() => conn.profiles.some((item) => item.id === form.value.id))

const filteredProfiles = computed(() => {
  const term = keyword.value.trim().toLowerCase()
  if (!term) return conn.profiles
  return conn.profiles.filter((profile) =>
    [profile.name, profile.host, profile.user].join(' ').toLowerCase().includes(term)
  )
})

function initial(name) {
  return (name || '?').trim().charAt(0).toUpperCase()
}

function sessionOf(profile) {
  return sessions.sessions.find((item) => item.profileId === profile.id) ?? null
}

function openCreate() {
  form.value = conn.createProfile()
  dialogVisible.value = true
}

function openEdit(profile) {
  form.value = { ...profile }
  dialogVisible.value = true
}

function save() {
  if (!form.value.host?.trim() || !form.value.user?.trim()) {
    toast.add({ severity: 'warn', summary: '请填写主机地址和用户名', life: 2600 })
    return
  }
  const saved = conn.saveProfile({ ...form.value })
  form.value = { ...saved }
  dialogVisible.value = false
  toast.add({ severity: 'success', summary: '已保存', detail: saved.name, life: 1800 })
}

function removeProfile(profile) {
  if (!window.confirm(`删除服务器「${profile.name}」？`)) return
  conn.removeProfile(profile.id)
}

async function connectProfile(profile) {
  if (connectingId.value) return
  connectingId.value = profile.id
  try {
    const sessionId = await connect(profile)
    const session = {
      sessionId,
      profileId: profile.id,
      name: profile.name,
      host: profile.host,
      user: profile.user,
    }
    sessions.add(session)
    await SSHService.OpenSessionWindow(sessionId, 'desktop', `${profile.name} — 远程桌面`, profile.host)
    toast.add({ severity: 'success', summary: '连接成功', detail: `${profile.user}@${profile.host}`, life: 2200 })
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: '连接失败',
      detail: String(error?.message ?? error),
      life: 5000,
    })
  } finally {
    connectingId.value = null
  }
}

async function openWindow(session) {
  try {
    await SSHService.OpenSessionWindow(session.sessionId, 'desktop', `${session.name} — 远程桌面`, session.host)
  } catch (error) {
    toast.add({ severity: 'error', summary: '打开窗口失败', detail: String(error?.message ?? error), life: 4000 })
  }
}

async function disconnectSession(session) {
  try {
    await disconnect(session.sessionId)
  } catch (error) {
    // The session may already be gone; ssh:closed also removes it from the store.
  }
  sessions.remove(session.sessionId)
}
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="logo"><i class="fa-solid fa-server"></i></span>
        <span>
          OpenDesk
          <span class="sub">SSH WORKSPACE</span>
        </span>
      </div>

      <button type="button" class="nav-item" :class="{ active: section === 'servers' }" @click="section = 'servers'">
        <i class="fa-solid fa-hard-drive"></i>
        服务器
        <span class="count">{{ conn.profiles.length }}</span>
      </button>
      <button type="button" class="nav-item" :class="{ active: section === 'sessions' }" @click="section = 'sessions'">
        <i class="fa-solid fa-tower-broadcast"></i>
        活动会话
        <span class="count">{{ sessions.sessions.length }}</span>
      </button>

      <div class="side-footer">
        连接信息保存在本机<br />
        密码为明文存储
      </div>
    </aside>

    <main class="main">
      <template v-if="section === 'servers'">
        <header class="main-head">
          <div>
            <h2>服务器</h2>
            <p>点击服务器卡片即可连接，并为该会话打开一个独立的远程桌面窗口。</p>
          </div>
          <div class="head-actions">
            <InputText v-model="keyword" placeholder="搜索名称或 IP" size="small" style="width: 180px" />
            <Button label="添加服务器" icon="pi pi-plus" size="small" @click="openCreate" />
          </div>
        </header>

        <div class="content">
          <div v-if="conn.profiles.length === 0" class="panel-empty">
            <span class="glyph"><i class="fa-solid fa-server"></i></span>
            <div>
              <div style="color: var(--od-text); font-weight: 600; margin-bottom: 4px">还没有添加服务器</div>
              <div>添加一台远程 Linux 主机，密码或 SSH 私钥都可以。</div>
            </div>
            <Button label="添加服务器" icon="pi pi-plus" size="small" @click="openCreate" />
          </div>

          <div v-else-if="filteredProfiles.length === 0" class="panel-empty">
            <span class="glyph"><i class="fa-solid fa-magnifying-glass"></i></span>
            <div>没有匹配「{{ keyword }}」的服务器</div>
          </div>

          <div v-else class="server-grid">
            <article
              v-for="profile in filteredProfiles"
              :key="profile.id"
              class="server-card"
              :title="`连接 ${profile.user}@${profile.host}`"
              @click="connectProfile(profile)"
            >
              <span class="avatar" :style="{ background: `linear-gradient(145deg, ${profile.color}, ${profile.color}99)` }">
                {{ initial(profile.name) }}
              </span>
              <div>
                <div class="name">{{ profile.name }}</div>
                <div class="target">{{ profile.user }}@{{ profile.host }}:{{ profile.port }}</div>
              </div>

              <div class="card-foot">
                <span v-if="connectingId === profile.id" class="auth">
                  <i class="pi pi-spin pi-spinner"></i> 连接中…
                </span>
                <span v-else-if="sessionOf(profile)" class="od-badge online">
                  <i class="fa-solid fa-circle" style="font-size: 6px"></i> 已连接
                </span>
                <span v-else class="auth">
                  <i class="fa-solid fa-key"></i>
                  {{ profile.privateKeyPath ? '私钥' : '密码' }}
                </span>

                <span class="card-actions" @click.stop>
                  <Button icon="pi pi-pencil" size="small" text severity="secondary" @click="openEdit(profile)" />
                  <Button icon="pi pi-trash" size="small" text severity="danger" @click="removeProfile(profile)" />
                </span>
              </div>
            </article>

            <button type="button" class="add-card" @click="openCreate">
              <i class="fa-solid fa-circle-plus"></i>
              添加服务器
            </button>
          </div>
        </div>
      </template>

      <template v-else>
        <header class="main-head">
          <div>
            <h2>活动会话</h2>
            <p>每个会话对应一个远程桌面窗口，关闭该窗口时自动断开连接。</p>
          </div>
        </header>

        <div class="content">
          <div v-if="sessions.sessions.length === 0" class="panel-empty">
            <span class="glyph"><i class="fa-solid fa-tower-broadcast"></i></span>
            <div>当前没有活动会话，先到「服务器」点击一台主机连接。</div>
            <Button label="去连接服务器" icon="pi pi-arrow-right" size="small" text @click="section = 'servers'" />
          </div>

          <table v-else class="session-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>地址</th>
                <th>会话 ID</th>
                <th style="width: 220px">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="session in sessions.sessions" :key="session.sessionId">
                <td>{{ session.name }}</td>
                <td class="mono muted">{{ session.user }}@{{ session.host }}</td>
                <td class="mono muted">{{ session.sessionId }}</td>
                <td>
                  <div style="display: flex; gap: 6px">
                    <Button label="打开桌面" icon="pi pi-desktop" size="small" outlined @click="openWindow(session)" />
                    <Button label="断开" icon="pi pi-power-off" size="small" text severity="danger" @click="disconnectSession(session)" />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </main>

    <Dialog
      v-model:visible="dialogVisible"
      :header="isEditing ? '编辑服务器' : '添加服务器'"
      modal
      :style="{ width: '460px' }"
      append-to="body"
    >
      <div class="server-form">
        <label>
          <span>名称</span>
          <InputText v-model="form.name" placeholder="生产服务器" fluid />
        </label>
        <div class="row">
          <label class="grow">
            <span>主机地址</span>
            <InputText v-model="form.host" placeholder="192.168.1.10 或 example.com" fluid />
          </label>
          <label style="width: 110px">
            <span>端口</span>
            <InputText v-model="form.port" type="number" placeholder="22" fluid />
          </label>
        </div>
        <label>
          <span>用户名</span>
          <InputText v-model="form.user" placeholder="root" fluid />
        </label>
        <label>
          <span>密码</span>
          <Password v-model="form.password" :feedback="false" toggleMask fluid placeholder="可留空，改用私钥" />
        </label>
        <label>
          <span>私钥路径（可选）</span>
          <InputText v-model="form.privateKeyPath" placeholder="C:\Users\me\.ssh\id_rsa" fluid />
        </label>
        <label>
          <span>私钥密码（可选）</span>
          <Password v-model="form.passphrase" :feedback="false" toggleMask fluid />
        </label>
      </div>

      <template #footer>
        <Button label="取消" text severity="secondary" size="small" @click="dialogVisible = false" />
        <Button label="保存" icon="pi pi-check" size="small" @click="save" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.server-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 6px;
}

.server-form label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  color: var(--od-text-dim);
}

.server-form .row {
  display: flex;
  gap: 10px;
}

.server-form .grow {
  flex: 1;
}
</style>
