// --- Color utilities ---
function thresholdColor(pct) {
    if (pct >= 90) return '#ff5555';
    if (pct >= 70) return '#ffaf5f';
    if (pct >= 30) return '#ffd700';
    return '#00ff87';
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

// --- Progress Bar ---
function renderProgressBar(containerId, label, percent, used, total) {
    var container = document.getElementById(containerId);
    var barWidth = 200;
    var color = thresholdColor(percent);

    var html = '<span class="progress-label">' + label + '</span>' +
        '<span class="progress-track" style="width:' + barWidth + 'px">' +
        '<span class="progress-fill" style="width:' + percent + '%;background:' + color + '"></span>' +
        '</span>' +
        '<span class="progress-pct" style="color:' + color + '">' + percent.toFixed(1) + '%</span>';

    if (used !== undefined && total !== undefined) {
        html += '<div class="progress-detail">' + formatBytes(used) + ' / ' + formatBytes(total) + '</div>';
    }

    container.innerHTML = html;
}

// --- Sparkline (Canvas) ---
function renderSparkline(canvasId, data, maxPoints) {
    var canvas = document.getElementById(canvasId);
    var ctx = canvas.getContext('2d');
    var dpr = window.devicePixelRatio || 1;

    var rect = canvas.getBoundingClientRect();
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    var w = rect.width;
    var h = rect.height;
    var pad = 2;

    ctx.clearRect(0, 0, w, h);

    var points = data.slice(-(maxPoints || 100));
    if (points.length < 2) return;

    var stepX = (w - pad * 2) / (points.length - 1);

    // Fill gradient
    var gradient = ctx.createLinearGradient(0, 0, 0, h);
    gradient.addColorStop(0, 'rgba(0, 255, 135, 0.3)');
    gradient.addColorStop(1, 'rgba(0, 255, 135, 0.02)');

    ctx.beginPath();
    ctx.moveTo(pad, h - pad);
    for (var i = 0; i < points.length; i++) {
        var x = pad + i * stepX;
        var y = h - pad - (points[i] / 100) * (h - pad * 2);
        ctx.lineTo(x, y);
    }
    ctx.lineTo(pad + (points.length - 1) * stepX, h - pad);
    ctx.closePath();
    ctx.fillStyle = gradient;
    ctx.fill();

    // Line
    ctx.beginPath();
    for (var i = 0; i < points.length; i++) {
        var x = pad + i * stepX;
        var y = h - pad - (points[i] / 100) * (h - pad * 2);
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
    }
    ctx.strokeStyle = '#00ff87';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    // Latest point dot
    var lastX = pad + (points.length - 1) * stepX;
    var lastY = h - pad - (points[points.length - 1] / 100) * (h - pad * 2);
    ctx.beginPath();
    ctx.arc(lastX, lastY, 3, 0, Math.PI * 2);
    ctx.fillStyle = '#00ff87';
    ctx.fill();
}

// --- Cores Bar ---
function renderCores(containerId, perCore) {
    var container = document.getElementById(containerId);
    var maxHeight = 32;
    var sum = 0;
    for (var i = 0; i < perCore.length; i++) sum += perCore[i];
    var avg = sum / perCore.length;

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

    html += '<div class="core-col" style="margin-left:12px">' +
        '<span class="core-pct color-muted">avg</span>' +
        '<span class="core-pct" style="color:' + thresholdColor(avg) + '">' + avg.toFixed(1) + '%</span>' +
        '</div>';

    container.innerHTML = html;
}
