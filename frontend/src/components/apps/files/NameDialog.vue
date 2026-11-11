<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '新建文件夹' },
  label: { type: String, default: '' },
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  confirmText: { type: String, default: '确定' },
})

const emit = defineEmits(['update:visible', 'update:modelValue', 'confirm', 'cancel'])

const inputRef = ref(null)
const error = ref('')

watch(
  () => props.visible,
  async (visible) => {
    if (!visible) return
    error.value = ''
    await nextTick()
    inputRef.value?.focus()
    inputRef.value?.select()
  }
)

function submit() {
  const value = String(props.modelValue ?? '').trim()
  if (!value) {
    error.value = '名称不能为空'
    inputRef.value?.focus()
    return
  }
  if (value.includes('/')) {
    error.value = '名称不能包含 /'
    inputRef.value?.focus()
    return
  }
  emit('confirm', value)
}

function cancel() {
  error.value = ''
  emit('cancel')
}
</script>

<template>
  <Transition name="dlg-fade">
    <div v-if="visible" class="od-dialog-mask" @click.self="cancel" @keydown.esc="cancel">
      <div class="od-dialog" role="dialog" aria-modal="true">
        <header class="od-dialog-head">
          <span class="od-dialog-icon"><i class="fa-regular fa-folder"></i></span>
          <div>
            <div class="od-dialog-title">{{ title }}</div>
            <div v-if="label" class="od-dialog-sub">{{ label }}</div>
          </div>
        </header>

        <div class="od-dialog-body">
          <input
            ref="inputRef"
            class="od-dialog-input"
            :class="{ invalid: !!error }"
            :value="modelValue"
            :placeholder="placeholder"
            spellcheck="false"
            autocomplete="off"
            @input="emit('update:modelValue', $event.target.value)"
            @keydown.enter.prevent="submit"
            @keydown.esc.prevent="cancel"
          />
          <p v-if="error" class="od-dialog-error">{{ error }}</p>
        </div>

        <footer class="od-dialog-foot">
          <button type="button" class="od-btn" @click="cancel">取消</button>
          <button type="button" class="od-btn primary" @click="submit">{{ confirmText }}</button>
        </footer>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.od-dialog-mask {
  position: absolute;
  inset: 0;
  z-index: 9800;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(4, 7, 12, 0.55);
  backdrop-filter: blur(3px);
}

.od-dialog {
  width: 360px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: linear-gradient(180deg, #1b2331 0%, #161d29 100%);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}

.od-dialog-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 20px 12px;
}

.od-dialog-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  font-size: 15px;
  color: #fbbf24;
  background: rgba(251, 191, 36, 0.14);
  border: 1px solid rgba(251, 191, 36, 0.28);
}

.od-dialog-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--od-text);
}

.od-dialog-sub {
  margin-top: 2px;
  font-size: 11.5px;
  color: var(--od-text-dim);
}

.od-dialog-body {
  padding: 4px 20px 0;
}

.od-dialog-input {
  width: 100%;
  height: 36px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(0, 0, 0, 0.35);
  color: var(--od-text);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.od-dialog-input::placeholder {
  color: rgba(147, 161, 184, 0.6);
}

.od-dialog-input:focus {
  border-color: rgba(79, 140, 255, 0.75);
  box-shadow: 0 0 0 3px rgba(79, 140, 255, 0.18);
}

.od-dialog-input.invalid {
  border-color: rgba(239, 68, 68, 0.7);
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.14);
}

.od-dialog-error {
  margin: 8px 0 0;
  font-size: 11.5px;
  color: #fca5a5;
}

.od-dialog-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 18px 20px;
}

.od-btn {
  min-width: 76px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
  color: var(--od-text);
  font-size: 12.5px;
  cursor: pointer;
  transition: background 0.14s ease, border-color 0.14s ease;
}

.od-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.od-btn.primary {
  border-color: transparent;
  background: linear-gradient(135deg, #4f8cff, #2563eb);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 6px 18px rgba(37, 99, 235, 0.35);
}

.od-btn.primary:hover {
  background: linear-gradient(135deg, #629bff, #2f6ff0);
}

.dlg-fade-enter-active,
.dlg-fade-leave-active {
  transition: opacity 0.16s ease;
}

.dlg-fade-enter-active .od-dialog,
.dlg-fade-leave-active .od-dialog {
  transition: transform 0.16s ease;
}

.dlg-fade-enter-from,
.dlg-fade-leave-to {
  opacity: 0;
}

.dlg-fade-enter-from .od-dialog,
.dlg-fade-leave-to .od-dialog {
  transform: translateY(8px) scale(0.98);
}
</style>
