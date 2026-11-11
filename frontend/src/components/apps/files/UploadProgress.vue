<script setup>
import { computed } from 'vue'
import { useTransferStore } from '../../../stores/transfers'
import { formatSize } from '../../../utils/fileFormat'

const props = defineProps({
  sessionId: { type: String, default: '' },
})

const store = useTransferStore()

const visible = computed(() => store.active && (!store.sessionId || store.sessionId === props.sessionId))
const percent = computed(() => store.percent)
const indeterminate = computed(() => !store.current)
</script>

<template>
  <div v-if="visible" class="upload-progress">
    <div class="up-head">
      <i class="pi pi-spin pi-spinner up-spin"></i>
      <span class="up-name" :title="store.current?.name || ''">
        {{ store.current?.name || '准备上传…' }}
      </span>
      <span class="up-count">已完成 {{ store.done }}</span>
      <span v-if="store.failed" class="up-failed">失败 {{ store.failed }}</span>
      <span v-if="store.current" class="up-percent">{{ percent }}%</span>
    </div>

    <div class="up-track">
      <div
        class="up-fill"
        :class="{ indeterminate }"
        :style="indeterminate ? {} : { width: `${percent}%` }"
      ></div>
    </div>

    <div class="up-meta">
      <span v-if="store.current" class="mono">
        {{ formatSize(store.current.transferred) }} / {{ formatSize(store.current.size) }}
      </span>
      <span v-else>正在写入远程文件…</span>
      <span class="up-target mono" :title="store.remoteDir">→ {{ store.remoteDir }}</span>
    </div>
  </div>
</template>

<style scoped>
.upload-progress {
  flex: 0 0 auto;
  padding: 8px 12px 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(79, 140, 255, 0.06);
}

.up-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11.5px;
  color: var(--od-text-dim);
}

.up-spin {
  font-size: 11px;
  color: var(--od-accent);
}

.up-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--od-text);
  font-size: 12px;
}

.up-count {
  flex: 0 0 auto;
}

.up-failed {
  flex: 0 0 auto;
  color: #fca5a5;
}

.up-percent {
  flex: 0 0 auto;
  width: 38px;
  text-align: right;
  color: var(--od-accent);
  font-weight: 600;
}

.up-track {
  margin-top: 6px;
  height: 6px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
}

.up-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #4f8cff, #22d3ee);
  transition: width 0.18s ease;
}

.up-fill.indeterminate {
  width: 40%;
  animation: up-slide 1.1s ease-in-out infinite;
}

@keyframes up-slide {
  0% {
    margin-left: -40%;
  }
  100% {
    margin-left: 100%;
  }
}

.up-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 5px;
  font-size: 11px;
  color: var(--od-text-dim);
}

.up-target {
  max-width: 50%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
