// ==================== 像素级透明度查询 ====================
window.checkPixelTransparent = function(x, y) {
    if (!checkPixelTransparent._ctx) {
        checkPixelTransparent._cvs = document.createElement('canvas');
        checkPixelTransparent._cvs.width = 1;
        checkPixelTransparent._cvs.height = 1;
        checkPixelTransparent._ctx = checkPixelTransparent._cvs.getContext('2d');
    }
    var ctx = checkPixelTransparent._ctx;
    ctx.clearRect(0, 0, 1, 1);

    var canvases = document.querySelectorAll('canvas');
    for (var i = 0; i < canvases.length; i++) {
        var cv = canvases[i];
        if (cv.width <= 0 || cv.height <= 0) continue;
        var r = cv.getBoundingClientRect();
        if (x < r.left || x >= r.right || y < r.top || y >= r.bottom) continue;
        var dpr = (cv.width / r.width) || 1;
        var sx = Math.floor((x - r.left) * dpr);
        var sy = Math.floor((y - r.top) * dpr);
        if (sx < 0 || sy < 0 || sx >= cv.width || sy >= cv.height) continue;
        ctx.drawImage(cv, sx, sy, 1, 1, 0, 0, 1, 1);
    }

    var prevPE = document.body.style.pointerEvents;
    document.body.style.pointerEvents = 'auto';
    var el = document.elementFromPoint(x, y);
    document.body.style.pointerEvents = prevPE;
    if (el && el !== document.body && el !== document.documentElement && el.tagName !== 'CANVAS') {
        var cs = window.getComputedStyle(el);
        var opaque = false;
        var bg = cs.backgroundColor;
        var m = bg.match(/[\d.]+\)$/);
        if (m && parseFloat(m[0]) > 0.05) opaque = true;
        if (cs.backgroundImage !== 'none') opaque = true;
        if (parseFloat(cs.borderWidth) > 0 && cs.borderStyle !== 'none') opaque = true;
        var rects = el.getClientRects();
        for (var i = 0; i < rects.length; i++) {
            var tr = rects[i];
            if (x >= tr.left && x < tr.right && y >= tr.top && y < tr.bottom) { opaque = true; break; }
        }
        if (opaque) { ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, 1, 1); }
    }

    return ctx.getImageData(0, 0, 1, 1).data[3] < 5;
};

// ==================== 每帧逐点穿透检测（仅 Windows/WebView2） ====================
(function() {
    if (!/Edg/.test(navigator.userAgent)) return;

    window.__wtd_passthroughActive = false;
    var _lastTransparent = null, _debounceTimer = null, _pendingState = null;

    function tick() {
        requestAnimationFrame(tick);
        if (!window.__wtd_passthroughActive || !window.__lhpanda__) return;

        window.__lhpanda__('getCursorPos', {}).then(function(r) {
            if (!r || !r.success) return;
            var x = r.data.x, y = r.data.y;
            if (x < 0 || y < 0) return;

            var transparent = window.checkPixelTransparent(x, y);
            if (transparent === _lastTransparent) return;
            if (transparent === _pendingState) return;

            _pendingState = transparent;
            clearTimeout(_debounceTimer);
            _debounceTimer = setTimeout(function() {
                if (_pendingState !== _lastTransparent) {
                    _lastTransparent = _pendingState;
                    window.__lhpanda__('setInputPassthrough', { enabled: _lastTransparent });
                }
            }, 100);
        });
    }
    tick();
})();
