<script setup>
defineProps({
  visible: { type: Boolean, default: false },
})

const emit = defineEmits(['close'])

const groups = [
  {
    title: '浏览',
    items: [
      ['Enter', '打开目录 / 预览文件'],
      ['Backspace / Alt+←', '返回上级目录'],
      ['F5 / Ctrl+R', '刷新当前目录'],
      ['Ctrl+L', '编辑路径'],
    ],
  },
  {
    title: '选择',
    items: [
      ['↑ ↓', '上下切换选中'],
      ['Shift+↑ ↓', '连续多选'],
      ['Ctrl / Shift + 点击', '多选'],
      ['Ctrl+A', '全选'],
      ['Esc', '取消选择 / 关闭菜单'],
    ],
  },
  {
    title: '操作',
    items: [
      ['Ctrl+N', '新建文件夹'],
      ['Ctrl+U', '上传文件'],
      ['Ctrl+V', '粘贴复制的文件'],
      ['F2', '重命名'],
      ['Delete', '删除选中项'],
      ['右键', '文件快捷菜单'],
    ],
  },
]
</script>

<template>
  <div v-if="visible" class="shortcut-help" @click.stop>
    <header>
      <span><i class="fa-regular fa-keyboard"></i> 键盘快捷键</span>
      <button type="button" title="关闭" @click="emit('close')">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </header>
    <div class="shortcut-groups">
      <section v-for="group in groups" :key="group.title">
        <h5>{{ group.title }}</h5>
        <div v-for="[keys, text] in group.items" :key="keys" class="shortcut-row">
          <kbd>{{ keys }}</kbd>
          <span>{{ text }}</span>
        </div>
      </section>
    </div>
  </div>
</template>
