<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  crumbs: { type: Array, default: () => [] },
  editing: { type: Boolean, default: false },
  draft: { type: String, default: '' },
})

const emit = defineEmits(['navigate', 'start-edit', 'submit', 'cancel', 'update:draft'])

const inputRef = ref(null)

watch(
  () => props.editing,
  async (editing) => {
    if (!editing) return
    await nextTick()
    inputRef.value?.focus()
    inputRef.value?.select()
  }
)
</script>

<template>
  <div class="crumb-bar">
    <div v-if="!editing" class="crumbs" title="点击编辑路径（Ctrl+L）" @click="emit('start-edit')">
      <template v-for="(crumb, index) in crumbs" :key="crumb.path">
        <span v-if="index > 0" class="sep">/</span>
        <button type="button" class="crumb" @click.stop="emit('navigate', crumb.path)">{{ crumb.label }}</button>
      </template>
      <i class="fa-solid fa-pen-to-square edit-hint"></i>
    </div>

    <input
      v-else
      ref="inputRef"
      :value="draft"
      class="path-input mono"
      spellcheck="false"
      @input="emit('update:draft', $event.target.value)"
      @keydown.enter="emit('submit', $event.target.value)"
      @keydown.esc="emit('cancel')"
      @blur="emit('submit', $event.target.value)"
    />
  </div>
</template>

<style scoped>
.crumb-bar {
  flex: 0 0 auto;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.015);
}

.crumbs {
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 24px;
  font-size: 12px;
  overflow: hidden;
  cursor: text;
}

.crumb {
  border: 0;
  background: transparent;
  color: var(--od-text);
  padding: 2px 6px;
  border-radius: 5px;
  font-size: 12px;
  cursor: default;
  white-space: nowrap;
}

.crumb:hover {
  background: rgba(255, 255, 255, 0.09);
}

.crumbs .sep {
  color: var(--od-text-dim);
}

.edit-hint {
  margin-left: 8px;
  font-size: 10px;
  color: var(--od-text-dim);
  opacity: 0.5;
}

.path-input {
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border-radius: 6px;
  border: 1px solid rgba(79, 140, 255, 0.55);
  background: rgba(0, 0, 0, 0.35);
  color: var(--od-text);
  outline: none;
  font-size: 12px;
}
</style>
