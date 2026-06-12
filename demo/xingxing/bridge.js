// ———— 桥接 API + 密码保护 + 快捷键注册 ————
// 共享逻辑，所有效果页面通过 <script src="bridge.js"> 引用

var APP_PWD_KEY = 'app_password';

// DOM 引用（密码弹窗由各页面提供，ID 固定）
var pwdOverlay = document.getElementById('pwd-overlay');
var pwdInput = document.getElementById('pwd-input');
var pwdRemember = document.getElementById('pwd-remember-cb');
var pwdError = document.getElementById('pwd-error');
var pwdTitle = document.getElementById('pwd-title');
var pwdMsg = document.getElementById('pwd-msg');
var pwdInfo = document.getElementById('pwd-info');
var pwdConfirm = document.getElementById('pwd-confirm');
var pwdCancel = document.getElementById('pwd-cancel');

// 当前会话的有效密码（空=无密码），从 sessionStorage 恢复
var _sessionPwd = sessionStorage.getItem('_sessionPwd') || '';
// 当前弹窗模式: 'setup' | 'verify'
var _dialogMode = 'setup';

// ==================== 持久密码 ====================

function loadPersistPwd(callback) {
    if (!window.__lhpanda__) { callback(''); return; }
    window.__lhpanda__('getItem', { key: APP_PWD_KEY })
        .then(function(r) { callback(r && r.found ? r.value : ''); })
        .catch(function() { callback(''); });
}

function savePersistPwd(pwd, callback) {
    window.__lhpanda__('setItem', { key: APP_PWD_KEY, value: pwd })
        .then(function() { if (callback) callback(); })
        .catch(function() { if (callback) callback(); });
}

// ==================== 密码弹窗 ====================

function showDialog(mode) {
    _dialogMode = mode;
    if (mode === 'setup') {
        pwdTitle.textContent = '感谢使用';
        pwdMsg.textContent = '请输入关闭密码（每次启动设置）';
        pwdInfo.style.display = 'block';
        pwdRemember.style.display = '';
        pwdConfirm.textContent = '确认';
    } else { // verify
        pwdTitle.textContent = '🔒 程序已锁定';
        pwdMsg.textContent = '请输入密码以关闭程序';
        pwdInfo.style.display = 'none';
        pwdRemember.style.display = 'none';
        pwdConfirm.textContent = '确认';
    }
    pwdError.style.display = 'none';
    pwdInput.value = '';
    pwdRemember.checked = false;
    pwdOverlay.style.display = 'flex';
    setTimeout(function() { pwdInput.focus(); }, 100);
}

function hideDialog() {
    pwdOverlay.style.display = 'none';
}

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

    // 键盘阻止（输入框内允许输入）
    document.addEventListener('keydown', function(e) {
        if (e.target && e.target.tagName === 'INPUT') return;
        e.preventDefault();
    }, { capture: true });
    document.addEventListener('keyup', function(e) {
        if (e.target && e.target.tagName === 'INPUT') return;
        e.preventDefault();
    }, { capture: true });

    // 密码弹窗：仅 options.showPwd === true 时弹出（默认不弹）
    if (options && options.showPwd) {
        showDialog('setup');
    }

    // 效果页面的初始化回调
    if (options && options.ready) options.ready();
}

// ==================== 密码弹窗按钮事件 ====================

pwdConfirm.onclick = function() {
    var val = pwdInput.value.trim();
    if (!val) return;
    if (_dialogMode === 'setup') {
        if (pwdRemember.checked) {
            savePersistPwd(val, function() {
                _sessionPwd = val;
                sessionStorage.setItem('_sessionPwd', val);
                hideDialog();
            });
        } else {
            _sessionPwd = val;
            sessionStorage.setItem('_sessionPwd', val);
            hideDialog();
        }
    } else {
        if (val === _sessionPwd) {
            hideDialog();
            window.__lhpanda__('closeWindow', {});
        } else {
            pwdError.style.display = 'block';
            pwdInput.value = '';
            pwdInput.focus();
        }
    }
};

pwdCancel.onclick = function() {
    if (_dialogMode === 'setup') {
        _sessionPwd = '';
        sessionStorage.removeItem('_sessionPwd');
        hideDialog();
    } else {
        hideDialog();
    }
};

// Enter 提交
pwdInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter') pwdConfirm.click();
});

// ==================== 效果切换 ====================

var EFFECTS = [
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
            if (window.__lhpanda__) {
                if (_sessionPwd !== '') {
                    showDialog('verify');
                } else {
                    window.__lhpanda__('closeWindow', {});
                }
            }
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
