<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Button from 'primevue/button'
import { SSHService } from '../../services/ssh'

const props = defineProps({
  sessionId: { type: String, default: '' },
  host: { type: String, default: '' },
})

const sessionId = props.sessionId

const metrics = ref(null)
const error = ref('')
const paused = ref(false)
let timer = null

const memPercent = computed(() => {
  if (!metrics.value?.memTotal) return 0
  return (metrics.value.memUsed / metrics.value.memTotal) * 100
})

const swapPercent = computed(() => {
  if (!metrics.value?.swapTotal) return 0
  return (metrics.value.swapUsed / metrics.value.swapTotal) * 100
})

function tone(value) {
  if (value >= 85) return 'danger'
  if (value >= 60) return 'warn'
  return 'ok'
}

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${units[index]}`
}

async function refresh() {
  if (!sessionId) return
  try {
    metrics.value = await SSHService.Metrics(sessionId)
    error.value = ''
  } catch (err) {
    error.value = String(err?.message ?? err)
  }
}

function togglePause() {
  paused.value = !paused.value
  if (!paused.value) refresh()
}

onMounted(() => {
  refresh()
  timer = window.setInterval(() => {
    if (!paused.value) refresh()
  }, 3000)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="app-root">
    <div v-if="!sessionId" class="app-empty">
      <span class="glyph"><i class="fa-solid fa-chart-line"></i></span>
      <div>这个窗口没有关联的 SSH 会话。</div>
    </div>

    <template v-else>
      <div class="app-header">
        <span class="od-badge online">
          <i class="fa-solid fa-circle" style="font-size: 7px"></i>
          {{ metrics?.hostname || '连接中' }}
        </span>
        <span class="muted" style="font-size: 12px">{{ metrics?.kernel }}</span>
        <span class="spacer"></span>
        <span class="muted" style="font-size: 11.5px">每 3 秒刷新</span>
        <Button
          :label="paused ? '继续' : '暂停'"
          :icon="paused ? 'pi pi-play' : 'pi pi-pause'"
          size="small"
          text
          severity="secondary"
          @click="togglePause"
        />
        <Button icon="pi pi-refresh" size="small" text severity="secondary" @click="refresh" />
      </div>

      <div class="app-body">
        <p v-if="error" class="monitor-error">{{ error }}</p>

        <div class="stat-grid">
          <section class="stat-card">
            <header>
              <span>CPU</span>
              <strong>{{ (metrics?.cpuPercent ?? 0).toFixed(1) }}%</strong>
            </header>
            <div class="bar">
              <div class="fill" :class="tone(metrics?.cpuPercent ?? 0)" :style="{ width: `${Math.min(metrics?.cpuPercent ?? 0, 100)}%` }"></div>
            </div>
            <footer>
              <span>{{ metrics?.cpuCores ?? '—' }} 核</span>
              <span>负载 {{ metrics?.loadAvg || '—' }}</span>
            </footer>
          </section>

          <section class="stat-card">
            <header>
              <span>内存</span>
              <strong>{{ memPercent.toFixed(1) }}%</strong>
            </header>
            <div class="bar">
              <div class="fill" :class="tone(memPercent)" :style="{ width: `${Math.min(memPercent, 100)}%` }"></div>
            </div>
            <footer>
              <span>{{ formatBytes(metrics?.memUsed) }} / {{ formatBytes(metrics?.memTotal) }}</span>
            </footer>
          </section>

          <section class="stat-card">
            <header>
              <span>交换分区</span>
              <strong>{{ swapPercent.toFixed(1) }}%</strong>
            </header>
            <div class="bar">
              <div class="fill" :class="tone(swapPercent)" :style="{ width: `${Math.min(swapPercent, 100)}%` }"></div>
            </div>
            <footer>
              <span>
                {{ formatBytes(metrics?.swapUsed) }} / {{ formatBytes(metrics?.swapTotal) }}
              </span>
            </footer>
          </section>

          <section class="stat-card">
            <header>
              <span>运行时间</span>
            </header>
            <div class="uptime">{{ metrics?.uptime || '—' }}</div>
            <footer>
              <span>{{ metrics?.kernel }}</span>
            </footer>
          </section>
        </div>

        <h5 class="section-title">磁盘</h5>
        <div class="disk-list">
          <div v-for="disk in metrics?.disks ?? []" :key="disk.mount" class="disk-row">
            <div class="disk-head">
              <span class="mono">{{ disk.mount }}</span>
              <span class="muted mono">{{ disk.filesystem }}</span>
              <span class="muted">{{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</span>
              <strong>{{ disk.percent.toFixed(0) }}%</strong>
            </div>
            <div class="bar slim">
              <div class="fill" :class="tone(disk.percent)" :style="{ width: `${Math.min(disk.percent, 100)}%` }"></div>
            </div>
          </div>
          <p v-if="!metrics?.disks?.length" class="muted" style="font-size: 12px">没有磁盘数据</p>
        </div>

        <h5 class="section-title">进程（按 CPU 排序）</h5>
        <table class="proc-table">
          <thead>
            <tr>
              <th style="width: 70px">PID</th>
              <th style="width: 110px">用户</th>
              <th style="width: 70px">CPU</th>
              <th style="width: 70px">内存</th>
              <th>命令</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="proc in metrics?.processes ?? []" :key="proc.pid">
              <td class="mono">{{ proc.pid }}</td>
              <td>{{ proc.user }}</td>
              <td class="mono">{{ proc.cpu.toFixed(1) }}%</td>
              <td class="mono">{{ proc.mem.toFixed(1) }}%</td>
              <td class="mono command">{{ proc.command }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="app-footer">
        <span class="mono">{{ host }}</span>
        <span>{{ paused ? '已暂停刷新' : '实时' }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.03);
}

.stat-card header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  font-size: 11.5px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--od-text-dim);
}

.stat-card header strong {
  font-size: 18px;
  letter-spacing: 0;
  color: var(--od-text);
}

.stat-card footer {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 11.5px;
  color: var(--od-text-dim);
}

.uptime {
  font-size: 14px;
  font-weight: 600;
  color: var(--od-text);
}

.bar {
  height: 7px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
}

.bar.slim {
  height: 5px;
}

.fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.4s ease;
}

.fill.ok {
  background: linear-gradient(90deg, #22d3ee, #4f8cff);
}

.fill.warn {
  background: linear-gradient(90deg, #f59e0b, #f97316);
}

.fill.danger {
  background: linear-gradient(90deg, #f43f5e, #ef4444);
}

.section-title {
  margin: 4px 0 8px;
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--od-text-dim);
  font-weight: 600;
}

.disk-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 18px;
}

.disk-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.disk-head {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
}

.disk-head strong {
  margin-left: auto;
  font-size: 12px;
}

.proc-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.proc-table thead th {
  text-align: left;
  padding: 6px 8px;
  font-size: 11px;
  color: var(--od-text-dim);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  font-weight: 600;
}

.proc-table td {
  padding: 5px 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 420px;
}

.proc-table .command {
  color: var(--od-text-dim);
}

.monitor-error {
  margin: 0 0 12px;
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 12px;
  color: #fca5a5;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
}
</style>
