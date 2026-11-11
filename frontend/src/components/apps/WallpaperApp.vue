<script setup>
import { computed, ref } from 'vue'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import { useDesktopStore } from '../../stores/desktop'
import { WALLPAPERS, findWallpaper } from '../../config/wallpapers'

const desktop = useDesktopStore()
const toast = useToast()
const fileInput = ref(null)

// 与其它应用保持相同的窗口入参（壁纸本身不依赖会话）。
defineProps({
  sessionId: { type: String, default: '' },
  host: { type: String, default: '' },
})

const MAX_IMAGE_SIZE = 2 * 1024 * 1024

const currentPreset = computed(() => (desktop.wallpaper.image ? null : findWallpaper(desktop.wallpaper.id)))
const currentName = computed(() => (desktop.wallpaper.image ? '自定义图片' : currentPreset.value.name))

function pickImage() {
  fileInput.value?.click()
}

function onFileChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.add({ severity: 'warn', summary: '请选择图片文件', life: 2600 })
    return
  }
  if (file.size > MAX_IMAGE_SIZE) {
    toast.add({ severity: 'warn', summary: '图片太大', detail: '请选择小于 2MB 的图片', life: 3200 })
    return
  }

  const reader = new FileReader()
  reader.onload = () => {
    const persisted = desktop.applyWallpaperImage(String(reader.result))
    if (!persisted) {
      toast.add({
        severity: 'warn',
        summary: '已应用壁纸',
        detail: '本机存储空间不足，重启后可能恢复默认壁纸',
        life: 4000,
      })
    }
  }
  reader.onerror = () => {
    toast.add({ severity: 'error', summary: '读取图片失败', life: 3000 })
  }
  reader.readAsDataURL(file)
}

function choosePreset(preset) {
  desktop.applyWallpaper(preset.id)
}
</script>

<template>
  <div class="app-root wallpaper-app">
    <div class="app-header">
      <span class="wallpaper-heading">
        <i class="fa-regular fa-image"></i>
        桌面壁纸
      </span>
      <span class="spacer"></span>
      <Button label="上传图片" icon="pi pi-upload" size="small" outlined @click="pickImage" />
      <Button
        label="恢复默认"
        icon="pi pi-refresh"
        size="small"
        text
        severity="secondary"
        @click="desktop.resetWallpaper()"
      />
      <input ref="fileInput" type="file" accept="image/*" class="wallpaper-file" @change="onFileChange" />
    </div>

    <div class="app-body wallpaper-body">
      <p class="wallpaper-hint">选择内置壁纸，或上传本地图片（不超过 2MB，仅保存在本机）。</p>

      <div class="wallpaper-grid">
        <button
          v-for="preset in WALLPAPERS"
          :key="preset.id"
          type="button"
          class="wallpaper-card"
          :class="{ active: !desktop.wallpaper.image && desktop.wallpaper.id === preset.id }"
          @click="choosePreset(preset)"
        >
          <span class="wallpaper-thumb" :style="{ background: preset.background }"></span>
          <span class="wallpaper-name">
            {{ preset.name }}
            <i
              v-if="!desktop.wallpaper.image && desktop.wallpaper.id === preset.id"
              class="fa-solid fa-check"
            ></i>
          </span>
        </button>
      </div>

      <div v-if="desktop.wallpaper.image" class="wallpaper-custom">
        <span class="wallpaper-thumb" :style="{ background: `url(${desktop.wallpaper.image}) center / cover` }"></span>
        <div class="wallpaper-custom-text">
          <div>自定义图片</div>
          <div class="muted">当前已应用为桌面壁纸</div>
        </div>
        <Button label="移除" icon="pi pi-trash" size="small" text severity="danger" @click="desktop.resetWallpaper()" />
      </div>
    </div>

    <div class="app-footer">
      <span>壁纸设置保存在本机，对所有服务器桌面生效</span>
      <span class="mono">{{ currentName }}</span>
    </div>
  </div>
</template>
