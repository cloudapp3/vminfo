(function() {
    'use strict';

    var state = {
        snapshot: null,
        procSort: 'cpu',
        procFilter: ''
    };

    // Client-side rolling history for charts that vminfo only sends as the
    // latest sample (no time series). Capped to keep memory bounded.
    var netHistory = { down: [], up: [], max: 60 };
    function pushNetHistory(snap) {
        var net = snap.network || {};
        netHistory.down.push(net.total_download_sec || 0);
        netHistory.up.push(net.total_upload_sec || 0);
        while (netHistory.down.length > netHistory.max) netHistory.down.shift();
        while (netHistory.up.length > netHistory.max) netHistory.up.shift();
    }

    var dom = {
        hostname: document.getElementById('hostname'),
        wsStatus: document.getElementById('ws-status'),
        wsStatusText: document.getElementById('ws-status-text'),
        updateTime: document.getElementById('update-time'),
        systemInfo: document.getElementById('system-info'),
        diskioTable: document.getElementById('diskio-table'),
        cpuStats: document.getElementById('cpu-stats'),
        networkSummary: document.getElementById('network-summary'),
        networkConnections: document.getElementById('network-connections'),
        networkInterfaces: document.getElementById('network-interfaces'),
        healthSummary: document.getElementById('health-summary'),
        alertsBlock: document.getElementById('alerts-block'),
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

    // Re-render when the theme changes so data-driven colors update live.
    if (window.VMinfoTheme) {
        VMinfoTheme.onChange(function() {
            if (state.snapshot) handleSnapshot(state.snapshot);
        });
    }

    // --- Main update ---
    function handleSnapshot(snap) {
        state.snapshot = snap;
        pushNetHistory(snap);
        renderHeader(snap);
        renderSystem(snap.system, snap.processes ? snap.processes.total : 0);
        renderDiskIO(snap.disk);
        renderResources(snap);
        renderNetwork(snap);
        renderHealth(snap);
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
                '<td>' + escapeHtml(String(d.device || '')) + '</td>' +
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

        renderRingGauge('mem-ring', {
            percent: snap.memory.percent,
            color: thresholdColor(snap.memory.percent),
            label: 'Memory',
            sub: formatBytes(snap.memory.used) + ' / ' + formatBytes(snap.memory.total)
        });

        renderProgressBar('swap-bar', 'Swap', snap.memory.swap_percent, snap.memory.swap_used, snap.memory.swap_total);

        if (snap.disk.filesystems && snap.disk.filesystems.length > 0) {
            var fs = snap.disk.filesystems[0];
            renderProgressBar('disk-bar', 'Disk', fs.percent, fs.used, fs.total);
        } else {
            renderProgressBar('disk-bar', 'Disk', 0);
        }

        if (snap.cpu.history) {
            renderBarChart('cpu-sparkline', snap.cpu.history, 60);
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

        renderLineChart('net-chart', [
            { data: netHistory.down, color: themeColor('green', '#00ff9c'), fill: 0.18 },
            { data: netHistory.up, color: themeColor('pink', '#ff79c6'), fill: 0.0 }
        ]);

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

        renderNetworkConnections(net);

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

            if (!isActive) {
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
            var rowClass = isActive ? 'active' : 'idle';

            visibleRows += '<tr class="network-row ' + rowClass + '">' +
                '<td class="col-iface"><span class="iface-dot ' + (isActive ? 'active' : 'idle') + '">' + (isActive ? '●' : '○') + '</span><span class="iface-name">' + escapeHtml(truncateIfaceName(iface.name, showTotals ? 14 : 12)) + '</span></td>' +
                '<td class="col-ip"><span class="' + ipClass + '">' + escapeHtml(ip) + '</span></td>' +
                '<td class="col-rx net-down">&darr; ' + formatBytesPerSec(iface.download_sec || 0) + '</td>' +
                '<td class="col-tx net-up">&uarr; ' + formatBytesPerSec(iface.upload_sec || 0) + '</td>' +
                '<td class="col-total col-total-rx">' + formatBytes(iface.rx_bytes || 0) + '</td>' +
                '<td class="col-total col-total-tx">' + formatBytes(iface.tx_bytes || 0) + '</td>' +
                '</tr>';
        }

        if (foldedCount > 0) {
            visibleRows += '<tr class="network-row network-fold">' +
                '<td class="col-iface" colspan="4"><span class="iface-dot idle">○</span><span class="iface-name">' + foldedCount + ' idle interfaces</span></td>' +
                '<td class="col-total col-total-rx">' + formatBytes(foldedRx) + '</td>' +
                '<td class="col-total col-total-tx">' + formatBytes(foldedTx) + '</td>' +
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
                '</tr></thead>' +
                '<tbody>' + visibleRows + '</tbody>' +
            '</table>';
    }

    function renderNetworkConnections(net) {
        var states = net.tcp_states || {};
        var order = [
            ['ESTABLISHED', themeColor('cyan', '#22d3ee')],
            ['TIME_WAIT', themeColor('purple', '#a855f7')],
            ['SYN_RECV', themeColor('blue', '#4ea8ff')],
            ['CLOSE_WAIT', themeColor('orange', '#ff9d4d')],
            ['FIN_WAIT', themeColor('yellow', '#ffd24a')],
            ['LISTEN', themeColor('green', '#00ff9c')]
        ];

        var entries = [];
        var maxState = 1;
        for (var i = 0; i < order.length; i++) {
            var name = order[i][0];
            var value = states[name];
            if (value) {
                entries.push({ name: name, value: value, color: order[i][1] });
                if (value > maxState) maxState = value;
            }
        }

        var html = '';
        if (entries.length > 0) {
            html += '<div class="conn-bars">';
            for (var j = 0; j < entries.length; j++) {
                var e = entries[j];
                var pct = (e.value / maxState) * 100;
                html += '<div class="conn-row">' +
                    '<span class="conn-name">' + e.name + '</span>' +
                    '<span class="conn-track"><span class="conn-fill" style="width:' + pct.toFixed(1) + '%;background:' + e.color + '"></span></span>' +
                    '<span class="conn-val" style="color:' + e.color + '">' + e.value + '</span>' +
                    '</div>';
            }
            html += '</div>';
        } else {
            html += '<div class="color-muted">No connection state data</div>';
        }

        var meta = '<span class="metric-group"><span class="net-label">TCP</span><span class="metric-value">' + (net.tcp_connections || 0) + '</span></span>' +
            '<span class="metric-group"><span class="net-label">UDP</span><span class="metric-value">' + (net.udp_connections || 0) + '</span></span>';
        if (net.conntrack_max > 0) {
            var cp = (net.conntrack_count || 0) / net.conntrack_max * 100;
            meta += '<span class="metric-group"><span class="net-label">conntrack</span><span class="metric-value" style="color:' + thresholdColor(cp) + '">' + (net.conntrack_count || 0) + '/' + net.conntrack_max + ' (' + cp.toFixed(0) + '%)</span></span>';
        }
        html += '<div class="conn-meta">' + meta + '</div>';

        dom.networkConnections.innerHTML = html;
    }

    // --- Health (score ring in left rail, alerts in right rail) ---
    function renderHealth(snap) {
        var health = snap.health || { score: 100, warnings: [] };
        var warnings = health.warnings || [];
        var score = typeof health.score === 'number' ? health.score : 100;
        var hc = (window.VMinfoTheme && VMinfoTheme.colors) || {};
        var scoreColor = score >= 85 ? (hc.green || '#00ff9c') : (score >= 65 ? (hc.yellow || '#ffd24a') : (hc.red || '#ff5c6c'));
        var statusText = score >= 85 ? 'Excellent' : (score >= 65 ? 'Warning' : 'Critical');
        var topProcesses = ((snap.processes && snap.processes.list) || []).slice(0, 5);

        // Left rail: ring gauge + status + top CPU
        var leftHtml =
            '<div class="health-head">' +
                '<div class="gauge-block" id="health-ring"></div>' +
                '<div class="health-meta">' +
                    '<div class="health-title" style="color:' + scoreColor + '">' + statusText + '</div>' +
                    '<div class="health-subtitle">' + warnings.length + ' active warning' + (warnings.length === 1 ? '' : 's') + '</div>' +
                '</div>' +
            '</div>';

        if (topProcesses.length > 0) {
            leftHtml += '<div class="health-top"><span class="net-label">Top CPU</span>';
            for (var j = 0; j < topProcesses.length; j++) {
                var p = topProcesses[j];
                leftHtml += '<span class="health-proc">' +
                    escapeHtml(String(p.name || p.command || p.pid)) +
                    ' <span style="color:' + thresholdColor(p.cpu_percent || 0) + '">' +
                    Number(p.cpu_percent || 0).toFixed(1) + '%</span></span>';
            }
            leftHtml += '</div>';
        }
        dom.healthSummary.innerHTML = leftHtml;
        renderRingGauge('health-ring', { percent: score, color: scoreColor, value: score });

        // Right rail: alerts
        var alertsHtml = '';
        if (warnings.length === 0) {
            alertsHtml = '<div class="alerts-ok"><span class="alerts-dot"></span>All systems operational</div>';
        } else {
            alertsHtml = '<div class="alerts-list">';
            for (var i = 0; i < Math.min(warnings.length, 6); i++) {
                var w = warnings[i];
                var cls = w.level === 'critical' ? 'critical' : 'warning';
                alertsHtml += '<div class="alert-row ' + cls + '">' +
                    '<span class="alert-level">' + escapeHtml(String(w.level || 'warning')).toUpperCase() + '</span>' +
                    '<span class="alert-msg">' + escapeHtml(String(w.message || w.code || 'warning')) + '</span>' +
                    '</div>';
            }
            if (warnings.length > 6) {
                alertsHtml += '<div class="alert-more">+' + (warnings.length - 6) + ' more</div>';
            }
            alertsHtml += '</div>';
        }
        if (dom.alertsBlock) dom.alertsBlock.innerHTML = alertsHtml;
    }

    // --- Processes ---
    function renderProcesses(procs) {
        var totalCount = procs.total;
        var list = procs.list.slice();

        if (state.procFilter) {
            var filtered = [];
            for (var i = 0; i < list.length; i++) {
                var p = list[i];
                if (String(p.name || '').toLowerCase().indexOf(state.procFilter) !== -1 ||
                    String(p.command || '').toLowerCase().indexOf(state.procFilter) !== -1 ||
                    String(p.user || '').toLowerCase().indexOf(state.procFilter) !== -1 ||
                    String(p.pid).indexOf(state.procFilter) !== -1 ||
                    String(p.ppid || '').indexOf(state.procFilter) !== -1 ||
                    String(p.status || '').toLowerCase().indexOf(state.procFilter) !== -1) {
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

        var fragment = document.createDocumentFragment();
        for (var i = 0; i < list.length; i++) {
            var p = list[i];
            var cpuColor = thresholdColor(p.cpu_percent);
            var memColor = thresholdColor(p.mem_percent);
            var command = p.command || p.name || '';
            var row = document.createElement('tr');
            appendProcessCell(row, 'col-pid', p.pid);
            appendProcessCell(row, 'col-cpu', Number(p.cpu_percent || 0).toFixed(1), cpuColor);
            appendProcessCell(row, 'col-mem', Number(p.mem_percent || 0).toFixed(1), memColor);
            appendProcessCell(row, 'col-rss', formatBytes(p.rss));
            appendProcessCell(row, 'col-user', p.user || '—');
            appendProcessCell(row, 'col-status', p.status || '—');
            appendProcessCell(row, 'col-age', formatDuration(p.uptime || 0));
            appendProcessCell(row, 'col-name', p.name || '—', '', command);
            appendProcessCell(row, 'col-command', command || '—', '', command);
            fragment.appendChild(row);
        }
        while (dom.procTbody.firstChild) {
            dom.procTbody.removeChild(dom.procTbody.firstChild);
        }
        dom.procTbody.appendChild(fragment);
    }

    function appendProcessCell(row, className, value, color, title) {
        var cell = document.createElement('td');
        cell.className = className;
        cell.textContent = String(value);
        if (color) cell.style.color = color;
        if (title !== undefined) cell.title = String(title);
        row.appendChild(cell);
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

    function formatDuration(seconds) {
        seconds = Math.max(0, Number(seconds || 0));
        var days = Math.floor(seconds / 86400);
        seconds -= days * 86400;
        var hours = Math.floor(seconds / 3600);
        seconds -= hours * 3600;
        var minutes = Math.floor(seconds / 60);
        if (days > 0) return days + 'd ' + hours + 'h';
        if (hours > 0) return hours + 'h ' + minutes + 'm';
        if (minutes > 0) return minutes + 'm';
        return Math.floor(seconds) + 's';
    }

    function renderLoadValue(load, cores) {
        var value = typeof load === 'number' ? load : 0;
        return '<span class="load-value" style="color:' + loadColor(value, cores) + '">' + value.toFixed(2) + '</span>';
    }

    function renderLoadBar(load, cores) {
        return '<span class="load-bar-char" style="color:' + loadColor(load || 0, cores) + '">' + repeatChar(loadBarChar(load || 0, cores), 3) + '</span>';
    }

    function loadColor(load, cores) {
        var c = (window.VMinfoTheme && VMinfoTheme.colors) || {};
        var safeCores = cores || 1;
        var ratio = load / safeCores;
        if (ratio >= 1.0) return c.red || '#ff5c6c';
        if (ratio >= 0.8) return c.orange || '#ff9d4d';
        if (ratio >= 0.5) return c.yellow || '#ffd24a';
        return c.green || '#00ff9c';
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
