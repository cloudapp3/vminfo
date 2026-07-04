// --- Color utilities ---
// Palette is read from CSS variables (set per theme on <html>) so data-driven
// colors follow the active theme. VMinfoTheme is defined in theme.js.
function themeColor(name, fallback) {
    var c = (window.VMinfoTheme && VMinfoTheme.colors) || {};
    return c[name] || fallback;
}

function thresholdColor(pct) {
    if (pct >= 90) return themeColor('red', '#ff5c6c');
    if (pct >= 70) return themeColor('orange', '#ff9d4d');
    if (pct >= 30) return themeColor('yellow', '#ffd24a');
    return themeColor('green', '#00ff9c');
}

function hexToRgba(hex, alpha) {
    var h = String(hex || '#00ff9c').replace('#', '');
    if (h.length === 3) {
        h = h.charAt(0) + h.charAt(0) + h.charAt(1) + h.charAt(1) + h.charAt(2) + h.charAt(2);
    }
    var n = parseInt(h, 16);
    if (isNaN(n)) return 'rgba(0,255,156,' + alpha + ')';
    var r = (n >> 16) & 255;
    var g = (n >> 8) & 255;
    var b = n & 255;
    return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
}

// --- Format utilities ---
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    var units = ['B', 'K', 'M', 'G', 'T'];
    var k = 1024;
    var i = Math.floor(Math.log(bytes) / Math.log(k));
    var val = bytes / Math.pow(k, i);
    if (i === 0) return bytes + ' B';
    return val.toFixed(1) + units[i];
}

function formatBytesPerSec(bytes) {
    return formatBytes(bytes) + '/s';
}

// --- Canvas helper ---
function sizeCanvas(canvas) {
    var dpr = window.devicePixelRatio || 1;
    var rect = canvas.getBoundingClientRect();
    canvas.width = Math.round(rect.width * dpr);
    canvas.height = Math.round(rect.height * dpr);
    var ctx = canvas.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return ctx;
}

// --- Progress Bar ---
function renderProgressBar(containerId, label, percent, used, total) {
    var container = document.getElementById(containerId);
    if (!container) return;
    var safePercent = Math.max(0, Math.min(100, percent || 0));
    var color = thresholdColor(safePercent);
    var detailHTML = '<span class="progress-detail progress-detail-empty"></span>';

    if (used !== undefined && total !== undefined) {
        detailHTML = '<span class="progress-detail">' + formatBytes(used) + ' / ' + formatBytes(total) + '</span>';
    }

    container.innerHTML =
        '<div class="progress-head">' +
            '<span class="progress-label">' + label + '</span>' +
            '<span class="progress-pct" style="color:' + color + '">' + safePercent.toFixed(1) + '%</span>' +
        '</div>' +
        '<div class="progress-main">' +
            '<div class="progress-track">' +
                '<div class="progress-fill" style="width:' + safePercent + '%;background:' + color + '"></div>' +
            '</div>' +
            detailHTML +
        '</div>';
}

// --- Ring gauge (SVG) ---
function renderRingGauge(containerId, opts) {
    var el = document.getElementById(containerId);
    if (!el) return;
    var pct = Math.max(0, Math.min(100, opts.percent || 0));
    var color = opts.color || themeColor('green', '#00ff9c');
    var r = 42;
    var c = 2 * Math.PI * r;
    var offset = c * (1 - pct / 100);
    var center = opts.value !== undefined ? opts.value : (Math.round(pct * 10) / 10) + '%';

    el.innerHTML =
        '<div class="gauge">' +
            '<svg class="gauge-svg" viewBox="0 0 100 100">' +
                '<circle class="gauge-track" cx="50" cy="50" r="' + r + '"></circle>' +
                '<circle class="gauge-fill" cx="50" cy="50" r="' + r + '" style="stroke:' + color + ';stroke-dasharray:' + c.toFixed(2) + ';stroke-dashoffset:' + offset.toFixed(2) + '"></circle>' +
            '</svg>' +
            '<div class="gauge-pct" style="color:' + color + '">' + center + '</div>' +
        '</div>' +
        (opts.label ? '<div class="gauge-label">' + opts.label + '</div>' : '') +
        (opts.sub ? '<div class="gauge-sub">' + opts.sub + '</div>' : '');
}

// --- Vertical bar chart (canvas) ---
function renderBarChart(canvasId, data, maxPoints) {
    var canvas = document.getElementById(canvasId);
    if (!canvas) return;
    var ctx = sizeCanvas(canvas);
    var rect = canvas.getBoundingClientRect();
    var w = rect.width, h = rect.height;
    ctx.clearRect(0, 0, w, h);

    var points = (data || []).slice(-(maxPoints || 60));
    var n = points.length;
    if (n === 0) return;

    var gap = n > 40 ? 1 : 2;
    var bw = Math.max(1, (w - gap * (n - 1)) / n);
    var top = 2;

    for (var i = 0; i < n; i++) {
        var v = Math.max(0, Math.min(100, points[i]));
        var bh = (v / 100) * (h - top);
        ctx.fillStyle = thresholdColor(v);
        ctx.fillRect(i * (bw + gap), h - bh, bw, Math.max(1, bh));
    }
}

// --- Multi-series line chart (canvas) ---
function renderLineChart(canvasId, series) {
    var canvas = document.getElementById(canvasId);
    if (!canvas) return;
    var ctx = sizeCanvas(canvas);
    var rect = canvas.getBoundingClientRect();
    var w = rect.width, h = rect.height;
    ctx.clearRect(0, 0, w, h);

    var pad = 3;
    var maxVal = 1;
    var s, i;
    for (s = 0; s < series.length; s++) {
        var d = series[s].data;
        for (i = 0; i < d.length; i++) {
            if (d[i] > maxVal) maxVal = d[i];
        }
    }

    for (s = 0; s < series.length; s++) {
        var ser = series[s];
        var data = ser.data;
        if (data.length < 1) continue;
        var stepX = data.length > 1 ? (w - pad * 2) / (data.length - 1) : 0;
        var x, y;

        if (ser.fill) {
            ctx.beginPath();
            ctx.moveTo(pad, h - pad);
            for (i = 0; i < data.length; i++) {
                x = pad + i * stepX;
                y = h - pad - (data[i] / maxVal) * (h - pad * 2);
                ctx.lineTo(x, y);
            }
            ctx.lineTo(pad + (data.length - 1) * stepX, h - pad);
            ctx.closePath();
            ctx.fillStyle = hexToRgba(ser.color, ser.fill);
            ctx.fill();
        }

        ctx.beginPath();
        for (i = 0; i < data.length; i++) {
            x = pad + i * stepX;
            y = h - pad - (data[i] / maxVal) * (h - pad * 2);
            if (i === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        }
        ctx.strokeStyle = ser.color;
        ctx.lineWidth = 1.5;
        ctx.stroke();
    }
}

// --- Cores Bar ---
function renderCores(containerId, perCore) {
    var container = document.getElementById(containerId);
    if (!container) return;
    var maxHeight = 54;

    var html = '';
    for (var i = 0; i < perCore.length; i++) {
        var pct = perCore[i];
        var height = Math.max(4, (pct / 100) * maxHeight);
        var color = thresholdColor(pct);
        html += '<div class="core-col">' +
            '<span class="core-pct" style="color:' + color + '">' + pct.toFixed(0) + '%</span>' +
            '<div class="core-bar" style="height:' + height + 'px;background:' + color + '"></div>' +
            '<span class="core-label">' + i + '</span>' +
            '</div>';
    }

    container.innerHTML = html;
}
