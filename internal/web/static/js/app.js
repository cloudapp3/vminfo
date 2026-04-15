(function() {
    'use strict';

    var state = {
        snapshot: null,
        procSort: 'cpu',
        procFilter: ''
    };

    var dom = {
        hostname: document.getElementById('hostname'),
        wsStatus: document.getElementById('ws-status'),
        wsStatusText: document.getElementById('ws-status-text'),
        updateTime: document.getElementById('update-time'),
        systemInfo: document.getElementById('system-info'),
        diskioTable: document.getElementById('diskio-table'),
        cpuStats: document.getElementById('cpu-stats'),
        networkSummary: document.getElementById('network-summary'),
        networkInterfaces: document.getElementById('network-interfaces'),
        procCount: document.getElementById('proc-count'),
        procTbody: document.getElementById('proc-tbody'),
        procFilter: document.getElementById('proc-filter'),
        procSort: document.getElementById('proc-sort')
    };

    var ws = new VMInfoWebSocket(
        function(data) { handleSnapshot(data); },
        function(status) { updateWSStatus(status); }
    );

    dom.procFilter.addEventListener('input', function(e) {
        state.procFilter = e.target.value.toLowerCase();
        if (state.snapshot) renderProcesses(state.snapshot.processes);
    });

    dom.procSort.addEventListener('change', function(e) {
        state.procSort = e.target.value;
        if (state.snapshot) renderProcesses(state.snapshot.processes);
    });

    window.addEventListener('resize', function() {
        if (state.snapshot) {
            renderNetwork(state.snapshot);
        }
    });

    // --- Main update ---
    function handleSnapshot(snap) {
        state.snapshot = snap;
        renderHeader(snap);
        renderSystem(snap.system, snap.processes ? snap.processes.total : 0);
        renderDiskIO(snap.disk);
        renderResources(snap);
        renderNetwork(snap);
        renderProcesses(snap.processes);
    }

    // --- Header ---
    function renderHeader(snap) {
        dom.hostname.textContent = snap.system.hostname;
        dom.updateTime.textContent = new Date(snap.timestamp).toLocaleTimeString();
    }

    function updateWSStatus(status) {
        dom.wsStatus.className = 'status-dot ' + status;
        dom.wsStatusText.textContent = status.toUpperCase();
    }

    // --- System Panel ---
    function renderSystem(sys, procTotal) {
        var rows = [
            ['OS', sys.os],
            ['Kernel', sys.kernel],
            ['Arch', sys.arch],
            ['Host', sys.hostname],
            ['CPU', sys.cpu_model + ' (' + sys.cores + 'C)'],
            ['Uptime', sys.uptime_human],
            ['Procs', procTotal]
        ];
        var html = '';
        for (var i = 0; i < rows.length; i++) {
            html += '<tr><td>' + escapeHtml(String(rows[i][0])) + '</td><td>' + escapeHtml(String(rows[i][1])) + '</td></tr>';
        }
        dom.systemInfo.innerHTML = html;
    }

    // --- Disk I/O Panel ---
    function renderDiskIO(disk) {
        if (!disk.io || disk.io.length === 0) {
            dom.diskioTable.innerHTML = '<tr><td class="color-muted">No disk I/O data</td></tr>';
            return;
        }
        var html = '<thead><tr>' +
            '<th>DEVICE</th><th style="text-align:right">READ</th>' +
            '<th style="text-align:right">WRITE</th><th style="text-align:right">IOPS</th>' +
            '</tr></thead><tbody>';
        for (var i = 0; i < disk.io.length; i++) {
            var d = disk.io[i];
            var isIdle = d.read_bytes_sec === 0 && d.write_bytes_sec === 0 && d.iops === 0;
            var cls = isIdle ? ' class="color-muted"' : '';
            html += '<tr' + cls + '>' +
                '<td>' + d.device + '</td>' +
                '<td style="text-align:right">' + formatBytesPerSec(d.read_bytes_sec) + '</td>' +
                '<td style="text-align:right">' + formatBytesPerSec(d.write_bytes_sec) + '</td>' +
                '<td style="text-align:right">' + d.iops + '</td>' +
                '</tr>';
        }
        html += '</tbody>';
        dom.diskioTable.innerHTML = html;
    }

    // --- Resources Panel ---
    function renderResources(snap) {
        renderProgressBar('cpu-bar', 'CPU', snap.cpu.total_percent);
        renderProgressBar('mem-bar', 'Mem', snap.memory.percent, snap.memory.used, snap.memory.total);
        renderProgressBar('swap-bar', 'Swap', snap.memory.swap_percent, snap.memory.swap_used, snap.memory.swap_total);

        if (snap.disk.filesystems && snap.disk.filesystems.length > 0) {
            var fs = snap.disk.filesystems[0];
            renderProgressBar('disk-bar', 'Disk', fs.percent, fs.used, fs.total);
        } else {
            renderProgressBar('disk-bar', 'Disk', 0);
        }

        if (snap.cpu.history) {
            renderSparkline('cpu-sparkline', snap.cpu.history, 100);
        }

        dom.cpuStats.innerHTML =
            '<span class="stat-label">cur</span> ' +
            '<span class="stat-value" style="color:' + thresholdColor(snap.cpu.total_percent) + '">' + snap.cpu.total_percent.toFixed(1) + '%</span>' +
            '<span style="margin:0 8px">' +
            '<span class="stat-label">avg</span> ' +
            '<span class="stat-value" style="color:' + thresholdColor(snap.cpu.avg_percent) + '">' + snap.cpu.avg_percent.toFixed(1) + '%</span></span>' +
            '<span><span class="stat-label">max</span> ' +
            '<span class="stat-value" style="color:' + thresholdColor(snap.cpu.max_percent) + '">' + snap.cpu.max_percent.toFixed(1) + '%</span></span>';

        if (snap.cpu.per_core) {
            renderCores('cores-bar', snap.cpu.per_core);
        }
    }

    // --- Network Panel ---
    function renderNetwork(snap) {
        var net = snap.network || {};
        var load = snap.load || {};
        var cores = (snap.system && snap.system.cores) || 1;
        var interfaces = sortInterfaces(net.interfaces || []);
        var showTotals = window.innerWidth >= 1100;
        var compact = window.innerWidth < 760;
        var maxIdleVisible = compact ? 1 : 3;

        dom.networkSummary.innerHTML =
            '<div class="network-summary-main">' +
                '<div class="load-block">' +
                    '<div class="load-values">' +
                        '<span class="net-label">Load</span>' +
                        renderLoadValue(load.load1, cores) +
                        renderLoadValue(load.load5, cores) +
                        renderLoadValue(load.load15, cores) +
                    '</div>' +
                    '<div class="load-bars">' +
                        '<span></span>' +
                        renderLoadBar(load.load1, cores) +
                        renderLoadBar(load.load5, cores) +
                        renderLoadBar(load.load15, cores) +
                    '</div>' +
                    '<div class="load-times">' +
                        '<span></span><span>1m</span><span>5m</span><span>15m</span>' +
                    '</div>' +
                '</div>' +
                '<div class="network-metrics">' +
                    '<span class="metric-group"><span class="net-label">Net</span><span class="net-down">&darr; ' + formatBytesPerSec(net.total_download_sec || 0) + '</span><span class="net-up">&uarr; ' + formatBytesPerSec(net.total_upload_sec || 0) + '</span></span>' +
                    '<span class="metric-group"><span class="net-label">TCP</span><span class="metric-value">' + (net.tcp_connections || 0) + '</span><span class="net-label">UDP</span><span class="metric-value">' + (net.udp_connections || 0) + '</span></span>' +
                '</div>' +
            '</div>';

        if (interfaces.length === 0) {
            dom.networkInterfaces.innerHTML = '<div class="color-muted">No network interface data</div>';
            return;
        }

        var visibleRows = '';
        var idleVisible = 0;
        var foldedCount = 0;
        var foldedRx = 0;
        var foldedTx = 0;

        for (var i = 0; i < interfaces.length; i++) {
            var iface = interfaces[i];
            var isActive = iface.download_sec > 0 || iface.upload_sec > 0;
            var totalWarn = totalErrDrops(iface);

            if (!isActive && totalWarn === 0) {
                if (idleVisible >= maxIdleVisible && interfaces.length > maxIdleVisible + 1) {
                    foldedCount++;
                    foldedRx += iface.rx_bytes || 0;
                    foldedTx += iface.tx_bytes || 0;
                    continue;
                }
                idleVisible++;
            }

            var ip = iface.ipv4 || '—';
            var ipClass = ip !== '—' && !isPrivateIP(ip) ? 'ip-public' : 'ip-private';
            var warnHtml = totalWarn > 0
                ? '<span class="iface-warn">⚠ ' + ((iface.rx_errors || 0) + (iface.tx_errors || 0)) + ' errs ' + ((iface.rx_drops || 0) + (iface.tx_drops || 0)) + ' drops</span>'
                : '';
            var rowClass = isActive ? 'active' : 'idle';

            visibleRows += '<tr class="network-row ' + rowClass + '">' +
                '<td class="col-iface"><span class="iface-dot ' + (isActive ? 'active' : 'idle') + '">' + (isActive ? '●' : '○') + '</span><span class="iface-name">' + escapeHtml(truncateIfaceName(iface.name, showTotals ? 14 : 12)) + '</span></td>' +
                '<td class="col-ip"><span class="' + ipClass + '">' + escapeHtml(ip) + '</span></td>' +
                '<td class="col-rx net-down">&darr; ' + formatBytesPerSec(iface.download_sec || 0) + '</td>' +
                '<td class="col-tx net-up">&uarr; ' + formatBytesPerSec(iface.upload_sec || 0) + '</td>' +
                '<td class="col-total col-total-rx">' + formatBytes(iface.rx_bytes || 0) + '</td>' +
                '<td class="col-total col-total-tx">' + formatBytes(iface.tx_bytes || 0) + '</td>' +
                '<td class="col-warn">' + warnHtml + '</td>' +
                '</tr>';
        }

        if (foldedCount > 0) {
            visibleRows += '<tr class="network-row network-fold">' +
                '<td class="col-iface" colspan="4"><span class="iface-dot idle">○</span><span class="iface-name">' + foldedCount + ' idle interfaces</span></td>' +
                '<td class="col-total col-total-rx">' + formatBytes(foldedRx) + '</td>' +
                '<td class="col-total col-total-tx">' + formatBytes(foldedTx) + '</td>' +
                '<td class="col-warn"></td>' +
                '</tr>';
        }

        dom.networkInterfaces.innerHTML =
            '<table class="data-table network-table">' +
                '<thead><tr>' +
                    '<th class="col-iface">IFACE</th>' +
                    '<th class="col-ip">IP</th>' +
                    '<th class="col-rx">RX/s</th>' +
                    '<th class="col-tx">TX/s</th>' +
                    '<th class="col-total col-total-rx">TOTAL RX</th>' +
                    '<th class="col-total col-total-tx">TOTAL TX</th>' +
                    '<th class="col-warn">WARN</th>' +
                '</tr></thead>' +
                '<tbody>' + visibleRows + '</tbody>' +
            '</table>';
    }

    // --- Processes ---
    function renderProcesses(procs) {
        var totalCount = procs.total;
        var list = procs.list.slice();

        if (state.procFilter) {
            var filtered = [];
            for (var i = 0; i < list.length; i++) {
                var p = list[i];
                if (p.name.toLowerCase().indexOf(state.procFilter) !== -1 ||
                    p.user.toLowerCase().indexOf(state.procFilter) !== -1 ||
                    String(p.pid).indexOf(state.procFilter) !== -1) {
                    filtered.push(p);
                }
            }
            list = filtered;
        }

        var sortFns = {
            cpu: function(a, b) { return b.cpu_percent - a.cpu_percent; },
            mem: function(a, b) { return b.mem_percent - a.mem_percent; },
            pid: function(a, b) { return b.pid - a.pid; },
            name: function(a, b) { return a.name.localeCompare(b.name); }
        };
        list.sort(sortFns[state.procSort] || sortFns.cpu);

        if (list.length > 50) list = list.slice(0, 50);

        dom.procCount.textContent = '(' + list.length + ' shown / ' + totalCount + ' total)';

        var html = '';
        for (var i = 0; i < list.length; i++) {
            var p = list[i];
            var cpuColor = thresholdColor(p.cpu_percent);
            var memColor = thresholdColor(p.mem_percent);
            html += '<tr>' +
                '<td class="col-pid">' + p.pid + '</td>' +
                '<td class="col-cpu" style="color:' + cpuColor + '">' + p.cpu_percent.toFixed(1) + '</td>' +
                '<td class="col-mem" style="color:' + memColor + '">' + p.mem_percent.toFixed(1) + '</td>' +
                '<td class="col-rss">' + formatBytes(p.rss) + '</td>' +
                '<td class="col-user">' + escapeHtml(p.user) + '</td>' +
                '<td class="col-status">' + p.status + '</td>' +
                '<td class="col-name">' + escapeHtml(p.name) + '</td>' +
                '</tr>';
        }
        dom.procTbody.innerHTML = html;
    }

    function escapeHtml(str) {
        var div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    function sortInterfaces(items) {
        return items.slice().sort(function(a, b) {
            var aActive = (a.download_sec || 0) > 0 || (a.upload_sec || 0) > 0;
            var bActive = (b.download_sec || 0) > 0 || (b.upload_sec || 0) > 0;
            var aHighlight = aActive || totalErrDrops(a) > 0;
            var bHighlight = bActive || totalErrDrops(b) > 0;
            if (aHighlight !== bHighlight) return aHighlight ? -1 : 1;
            if (aActive !== bActive) return aActive ? -1 : 1;
            var aPri = ifacePriority(a.name || '');
            var bPri = ifacePriority(b.name || '');
            if (aPri !== bPri) return aPri - bPri;
            return (a.name || '').localeCompare(b.name || '');
        });
    }

    function ifacePriority(name) {
        if (name.indexOf('eth') === 0 || name.indexOf('en') === 0) return 0;
        if (name.indexOf('wl') === 0) return 1;
        if (name.indexOf('br') === 0) return 2;
        if (name.indexOf('docker') === 0) return 3;
        if (name.indexOf('veth') === 0) return 4;
        return 5;
    }

    function truncateIfaceName(name, maxLen) {
        if (!name || name.length <= maxLen) return name || '—';
        if (maxLen >= 10) return name.slice(0, maxLen - 5) + '…' + name.slice(-4);
        return name.slice(0, maxLen - 1) + '…';
    }

    function isPrivateIP(ip) {
        return /^10\./.test(ip) ||
            /^127\./.test(ip) ||
            /^192\.168\./.test(ip) ||
            /^172\.(1[6-9]|2\d|3[0-1])\./.test(ip);
    }

    function totalErrDrops(iface) {
        return (iface.rx_errors || 0) + (iface.tx_errors || 0) + (iface.rx_drops || 0) + (iface.tx_drops || 0);
    }

    function renderLoadValue(load, cores) {
        var value = typeof load === 'number' ? load : 0;
        return '<span class="load-value" style="color:' + loadColor(value, cores) + '">' + value.toFixed(2) + '</span>';
    }

    function renderLoadBar(load, cores) {
        return '<span class="load-bar-char" style="color:' + loadColor(load || 0, cores) + '">' + repeatChar(loadBarChar(load || 0, cores), 3) + '</span>';
    }

    function loadColor(load, cores) {
        var safeCores = cores || 1;
        var ratio = load / safeCores;
        if (ratio >= 1.0) return '#ff5555';
        if (ratio >= 0.8) return '#ffaf5f';
        if (ratio >= 0.5) return '#ffd700';
        return '#00ff87';
    }

    function loadBarChar(load, cores) {
        var safeCores = cores || 1;
        var ratio = load / safeCores;
        if (ratio >= 1.0) return '█';
        if (ratio >= 0.8) return '▆';
        if (ratio >= 0.5) return '▄';
        return '▁';
    }

    function repeatChar(ch, count) {
        return new Array(count + 1).join(ch);
    }

    // Initial REST fetch
    fetch('/api/v1/snapshot')
        .then(function(r) { return r.json(); })
        .then(function(data) { handleSnapshot(data); })
        .catch(function() { console.log('Initial fetch failed, waiting for WS'); });
})();
