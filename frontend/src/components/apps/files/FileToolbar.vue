<script setup>
import Button from 'primevue/button'

defineProps({
  busy: { type: Boolean, default: false },
  selectedCount: { type: Number, default: 0 },
  canDownload: { type: Boolean, default: false },
  canRename: { type: Boolean, default: false },
  filter: { type: String, default: '' },
  showHidden: { type: Boolean, default: false },
})

const emit = defineEmits([
  'up',
  'refresh',
  'mkdir',
  'upload',
  'download',
  'rename',
  'remove',
  'show-shortcuts',
  'update:filter',
  'update:show-hidden',
])
</script>

<template>
  <div class="app-header file-toolbar">
    <Button icon="pi pi-arrow-up" size="small" text severity="secondary" title="上级目录（Backspace）" @click="emit('up')" />
    <Button icon="pi pi-refresh" size="small" text severity="secondary" title="刷新（F5）" :loading="busy" @click="emit('refresh')" />
    <Button
      label="新建文件夹"
      icon="pi pi-folder-plus"
      size="small"
      text
      severity="secondary"
      title="新建文件夹（Ctrl+N）"
      @click="emit('mkdir')"
    />
    <Button
      label="上传"
      icon="pi pi-upload"
      size="small"
      text
      severity="secondary"
      title="上传（Ctrl+U）"
      @click="emit('upload')"
    />
    <span class="spacer"></span>

    <div class="file-filter">
      <i class="fa-solid fa-magnifying-glass"></i>
      <input
        :value="filter"
        type="text"
        placeholder="筛选当前目录"
        spellcheck="false"
        @input="emit('update:filter', $event.target.value)"
      />
      <button v-if="filter" type="button" class="file-filter-clear" title="清除筛选" @click="emit('update:filter', '')">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </div>

    <Button
      :icon="showHidden ? 'pi pi-eye' : 'pi pi-eye-slash'"
      size="small"
      text
      severity="secondary"
      :title="showHidden ? '隐藏点开头的文件' : '显示隐藏文件'"
      @click="emit('update:show-hidden', !showHidden)"
    />
    <Button
      icon="pi pi-question-circle"
      size="small"
      text
      severity="secondary"
      title="键盘快捷键"
      @click="emit('show-shortcuts')"
    />

    <Button
      label="下载"
      icon="pi pi-download"
      size="small"
      outlined
      :disabled="!canDownload"
      title="下载（Enter）"
      @click="emit('download')"
    />
    <Button
      label="重命名"
      icon="pi pi-pencil"
      size="small"
      outlined
      :disabled="!canRename"
      title="重命名（F2）"
      @click="emit('rename')"
    />
    <Button
      label="删除"
      icon="pi pi-trash"
      size="small"
      text
      severity="danger"
      :disabled="selectedCount === 0"
      title="删除（Delete）"
      @click="emit('remove')"
    />
  </div>
</template>
