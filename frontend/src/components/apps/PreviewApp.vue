<script setup>
import { computed, ref, watch } from 'vue'
import Button from 'primevue/button'
import { useDesktopStore } from '../../stores/desktop'
import { SSHService } from '../../services/ssh'
import { PREVIEW_MAX_BYTES, formatSize, previewKind, previewUrl } from '../../utils/preview'

const props = defineProps({
  sessionId: { type: String, default: '' },
  host: { type: String, default: '' },
})

const desktop = useDesktopStore()
const target = computed(() => desktop.previewTarget)
const kind = computed(() => (target.value ? previewKind(target.value.name) : ''))

const loading = ref(false)
const error = ref('')
const text = ref('')

const src = computed(() =>
  target.value && kind.value !== 'text' ? previewUrl(props.sessionId, target.value.path) : ''
)

const tooLarge = computed(() => (target.value?.size ?? 0) > PREVIEW_MAX_BYTES)

async function loadText(file) {
  text.value = ''
  error.value = ''
  if (!file) return
  if (tooLarge.value) {
    error.value = `文件大小 ${formatSize(file.size)}，超过 1MB 不支持查看，请下载后打开。`
    return
  }
  loading.value = true
  try {
    text.value = await SSHService.ReadTextFile(props.sessionId, file.path)
  } catch (err) {
    error.value = String(err?.message ?? err)
  } finally {
    loading.value = false
  }
}

watch(
  () => target.value,
  (file) => {
    if (kind.value === 'text') loadText(file)
    else {
      error.value = ''
      text.value = ''
    }
  },
  { immediate: true }
)

async function download() {
  if (!target.value) return
  try {
    await SSHService.Download(props.sessionId, target.value.path)
  } catch (err) {
    error.value = String(err?.message ?? err)
  }
}
</script>

<template>
  <div v-if="!target" class="app-empty">
    <span class="glyph"><i class="fa-regular fa-file-lines"></i></span>
    <div>在文件管理里双击文件即可预览。</div>
  </div>

  <div v-else class="app-root preview-app">
    <div class="app-header">
      <i class="preview-icon" :class="{
        'fa-solid fa-film': kind === 'video',
        'fa-solid fa-music': kind === 'audio',
        'fa-regular fa-image': kind === 'image',
        'fa-regular fa-file-pdf': kind === 'pdf',
        'fa-regular fa-file-lines': kind === 'text',
      }"></i>
      <span class="preview-name" :title="target.path">{{ target.name }}</span>
      <span class="muted mono preview-size">{{ formatSize(target.size) }}</span>
      <span class="spacer"></span>
      <Button label="下载" icon="pi pi-download" size="small" text severity="secondary" @click="download" />
    </div>

    <div class="app-body preview-body">
      <p v-if="error" class="preview-error">{{ error }}</p>

      <video v-else-if="kind === 'video'" class="preview-media" :src="src" controls autoplay playsinline></video>

      <audio v-else-if="kind === 'audio'" class="preview-audio" :src="src" controls autoplay></audio>

      <img v-else-if="kind === 'image'" class="preview-image" :src="src" :alt="target.name" />

      <iframe v-else-if="kind === 'pdf'" class="preview-frame" :src="src" :title="target.name"></iframe>

      <div v-else-if="kind === 'text'" class="preview-text-wrap">
        <div v-if="loading" class="preview-hint"><i class="pi pi-spin pi-spinner"></i> 正在读取…</div>
        <pre v-else class="preview-text">{{ text }}</pre>
      </div>

      <div v-else class="preview-hint">
        <i class="fa-regular fa-circle-question"></i>
        暂不支持在线查看该类型，请下载到本地打开。
      </div>
    </div>

    <div class="app-footer">
      <span class="mono preview-path">{{ target.path }}</span>
      <span class="mono">{{ host }}</span>
    </div>
  </div>
</template>
