// 远程桌面内的应用与桌面图标元数据，桌面窗口、任务栏、开始菜单共用同一份配置。
export const APP_META = {
  terminal: {
    title: '控制台',
    icon: 'fa-solid fa-terminal',
    width: 900,
    height: 560,
    from: '#4f8cff',
    to: '#2563eb',
  },
  files: {
    title: '文件管理',
    icon: 'fa-regular fa-folder-open',
    width: 940,
    height: 580,
    from: '#f59e0b',
    to: '#d97706',
  },
  monitor: {
    title: '系统监控',
    icon: 'fa-solid fa-chart-line',
    width: 900,
    height: 620,
    from: '#10b981',
    to: '#059669',
  },
  wallpaper: {
    title: '壁纸',
    icon: 'fa-regular fa-image',
    width: 800,
    height: 600,
    from: '#f472b6',
    to: '#db2777',
  },
  preview: {
    title: '预览',
    icon: 'fa-regular fa-file-lines',
    width: 900,
    height: 620,
    from: '#38bdf8',
    to: '#0284c7',
  },
}

// 桌面图标的默认顺序（按列排版），实际坐标由 stores/desktop.js 计算并持久化。
export const DESKTOP_ICONS = ['terminal', 'files', 'monitor', 'wallpaper']
