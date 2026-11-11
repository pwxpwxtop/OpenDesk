<script setup>
import { formatSize, formatTime, iconFor } from '../../../utils/fileFormat'

defineProps({
  entries: { type: Array, default: () => [] },
  selectedPaths: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  sortKey: { type: String, default: 'name' },
  sortDir: { type: Number, default: 1 },
})

const emit = defineEmits(['select', 'open', 'sort', 'context'])

const columns = [
  { key: 'name', label: '名称', style: '' },
  { key: 'size', label: '大小', style: 'width: 110px' },
  { key: 'modTime', label: '修改时间', style: 'width: 160px' },
  { key: null, label: '权限', style: 'width: 120px' },
]

function sortIcon(key, active, dir) {
  if (!key) return ''
  if (!active) return 'fa-solid fa-sort'
  return dir > 0 ? 'fa-solid fa-sort-up' : 'fa-solid fa-sort-down'
}
</script>

<template>
  <div class="file-area" @contextmenu.self.prevent="emit('context', null, $event)">
    <table class="file-table">
      <thead>
        <tr>
          <th style="width: 34px"></th>
          <th
            v-for="column in columns"
            :key="column.label"
            :style="column.style"
            :class="{ sortable: column.key }"
            @click="column.key && emit('sort', column.key)"
          >
            {{ column.label }}
            <i
              v-if="column.key"
              class="file-sort"
              :class="sortIcon(column.key, sortKey === column.key, sortDir)"
            ></i>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="entry in entries"
          :key="entry.path"
          :class="{ selected: selectedPaths.includes(entry.path) }"
          @click="emit('select', entry, $event)"
          @dblclick="emit('open', entry)"
          @contextmenu.prevent="emit('context', entry, $event)"
        >
          <td class="icon-cell">
            <i :class="iconFor(entry)" :style="{ color: entry.isDir ? '#fbbf24' : '#93a1b8' }"></i>
          </td>
          <td class="name-cell" :title="entry.name">{{ entry.name }}</td>
          <td class="muted mono">{{ entry.isDir ? '—' : formatSize(entry.size) }}</td>
          <td class="muted">{{ formatTime(entry.modTime) }}</td>
          <td class="muted mono">{{ entry.mode }}</td>
        </tr>
        <tr v-if="!loading && entries.length === 0">
          <td colspan="5" class="empty-row">空目录</td>
        </tr>
        <tr v-if="loading">
          <td colspan="5" class="empty-row">正在读取…</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.file-area {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}

.file-table thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  text-align: left;
  padding: 7px 10px;
  font-weight: 600;
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--od-text-dim);
  background: #131a26;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  white-space: nowrap;
}

.file-table thead th.sortable {
  cursor: pointer;
  user-select: none;
}

.file-table thead th.sortable:hover {
  color: var(--od-text);
}

.file-sort {
  margin-left: 5px;
  font-size: 9px;
  opacity: 0.65;
}

.file-table tbody tr {
  cursor: default;
}

.file-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.045);
}

.file-table tbody tr.selected {
  background: var(--od-accent-soft);
}

.file-table td {
  padding: 6px 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.icon-cell {
  text-align: center;
}

.name-cell {
  color: var(--od-text);
}

.empty-row {
  text-align: center;
  color: var(--od-text-dim);
  padding: 24px;
}
</style>
