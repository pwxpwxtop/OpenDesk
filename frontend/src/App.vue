<script setup>
import { onMounted } from 'vue'
import Toast from 'primevue/toast'
import { initSSHBus, onSessionClosed, onUploadResult } from './services/ssh'
import { useSessionStore } from './stores/sessions'
import { useTransferStore } from './stores/transfers'

onMounted(() => {
  initSSHBus()
  onSessionClosed((sessionId) => {
    useSessionStore().remove(sessionId)
  })
  // 上传进度事件在所有窗口里都能收到，交给 transfer store 汇总。
  const transfers = useTransferStore()
  onUploadResult((kind, payload) => transfers.handle(kind, payload))
})
</script>

<template>
  <router-view />
  <Toast position="top-right" />
</template>
