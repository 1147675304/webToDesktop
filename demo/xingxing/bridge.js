// ———— 桥接 API + 快捷键注册 ————
// 共享逻辑，所有效果页面通过 <script src="bridge.js"> 引用

// ==================== 快捷键与按键映射 ====================

function initBridge(options) {
    if (!window.__lhpanda__) { setTimeout(function() { initBridge(options); }, 200); return; }

    // 按键映射
    window.__lhpanda__('setKeyMapping', { key: 'Super_L', mappedName: 'Win' });
    window.__lhpanda__('setKeyMapping', { key: 'Super_R', mappedName: 'Win' });
    window.__lhpanda__('setKeyMapping', { key: 'Alt_L', mappedName: 'Alt' });
    window.__lhpanda__('setKeyMapping', { key: 'Alt_R', mappedName: 'Alt' });
    window.__lhpanda__('setKeyMapping', { key: 'Control_L', mappedName: 'Ctrl' });
    window.__lhpanda__('setKeyMapping', { key: 'Control_R', mappedName: 'Ctrl' });

    // 注册 Alt+W 关闭
    window.__lhpanda__('registerShortcut', { keys: ['Alt+W'] });

    // 注册效果切换快捷键（在效果页面生效）
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Right'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Left'] });

    // ========== 系统级快捷键拦截（防逃逸） ==========

    // 窗口管理
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Tab'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+F4'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Esc'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+F7'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+F8'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+F10'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Space'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+A'] });
    window.__lhpanda__('registerShortcut', { keys: ['F11'] });

    // 任务管理器/系统工具
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Esc'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+Esc'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+F2'] });

    // Super/Win 键组合
    window.__lhpanda__('registerShortcut', { keys: ['Win'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+D'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+E'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+L'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+R'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+S'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+Tab'] });
    window.__lhpanda__('registerShortcut', { keys: ['Win+,'] });

    // 工作区切换
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Alt+Down'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Alt+Up'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Alt+Left'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Alt+Right'] });

    // 截图
    window.__lhpanda__('registerShortcut', { keys: ['Print'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Print'] });
    window.__lhpanda__('registerShortcut', { keys: ['Shift+Print'] });

    // ========== 浏览器级快捷键拦截（防逃逸） ==========

    // 刷新/导航
    window.__lhpanda__('registerShortcut', { keys: ['F5'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+R'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+R'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Left'] });
    window.__lhpanda__('registerShortcut', { keys: ['Alt+Right'] });

    // 开发者工具
    window.__lhpanda__('registerShortcut', { keys: ['F12'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+I'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+J'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+U'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+C'] });

    // 标签/窗口操作
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+T'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+N'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+N'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+W'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+W'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+T'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Tab'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+Tab'] });

    // 文件/打印
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+S'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+P'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+Shift+S'] });

    // 查找/帮助
    window.__lhpanda__('registerShortcut', { keys: ['F1'] });
    window.__lhpanda__('registerShortcut', { keys: ['F3'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+F'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+H'] });
    window.__lhpanda__('registerShortcut', { keys: ['Ctrl+J'] });

    // 键盘阻止（输入框内允许输入）
    document.addEventListener('keydown', function(e) {
        if (e.target && e.target.tagName === 'INPUT') return;
        e.preventDefault();
    }, { capture: true });
    document.addEventListener('keyup', function(e) {
        if (e.target && e.target.tagName === 'INPUT') return;
        e.preventDefault();
    }, { capture: true });

    // 输入穿透控制（Windows: WS_EX_TRANSPARENT，Linux: input shape）
    // options.inputPassthrough:
    //   true  → 鼠标事件穿透到下层窗口（纯视觉叠加层，如效果页面）
    //   false → 本窗口捕获所有鼠标事件（有交互元素的页面，如启动器）
    //   不传  → 不改变当前穿透状态
    if (options && typeof options.inputPassthrough === 'boolean') {
        window.__wtd_passthroughActive = options.inputPassthrough;
        window.__lhpanda__('setInputPassthrough', { enabled: options.inputPassthrough });
    }

    // 效果页面的初始化回调
    if (options && options.ready) options.ready();
}

// ==================== 效果切换 ====================

var EFFECTS = [
    { file: 'stars.html',        name: '✨ 星空' },
    { file: 'solar-system.html', name: '🌌 太阳系' },
    { file: 'galaxy.html',       name: '✨ 粒子星系' },
    { file: 'aurora.html',       name: '🌌 极光' },
    { file: 'matrix.html',       name: '💚 数字雨' }
];

function navigateEffect(dir) {
    var path = window.location.pathname;
    var m = path.match(/\/effects\/([^/]+)$/);
    if (!m) return;
    var current = m[1];
    for (var i = 0; i < EFFECTS.length; i++) {
        if (EFFECTS[i].file === current) {
            var next = (i + dir + EFFECTS.length) % EFFECTS.length;
            window.location.href = '/effects/' + EFFECTS[next].file;
            return;
        }
    }
}

// ==================== 全局快捷键事件 ====================

window.addEventListener('keyboard-shortcut', function (e) {
    if (!e.detail) return;
    switch (e.detail.key) {
        case 'Alt+W':
            if (window.__lhpanda__) window.__lhpanda__('closeWindow', {});
            break;
        case 'Ctrl+Right':
            navigateEffect(1);
            break;
        case 'Ctrl+Left':
            navigateEffect(-1);
            break;
    }
});

// 阻止选择/拖拽
document.addEventListener('selectstart', function(e) { e.preventDefault(); }, { capture: true });
document.addEventListener('dragstart', function(e) { e.preventDefault(); }, { capture: true });
