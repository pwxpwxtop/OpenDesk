<script setup>
defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  items: { type: Array, default: () => [] },
})

const emit = defineEmits(['select'])
</script>

<template>
  <div v-if="visible" class="context-menu" :style="{ left: `${x}px`, top: `${y}px` }" @click.stop>
    <template v-for="(item, index) in items" :key="index">
      <div v-if="item.separator" class="ctx-sep"></div>
      <button
        v-else
        type="button"
        class="ctx-item"
        :class="{ danger: item.danger }"
        @click="emit('select', item.action)"
      >
        <i :class="item.icon"></i>
        {{ item.label }}
      </button>
    </template>
  </div>
</template>
