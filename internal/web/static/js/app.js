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

    // --- Main update ---
    function handleSnapshot(snap) {
        state.snapshot = snap;
        renderHeader(snap);
        renderSystem(snap.system);
        renderDiskIO(snap.disk);
        renderResources(snap);
        renderNetwork(snap.network, snap.load);
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
    function renderSystem(sys) {
        var rows = [
            ['OS', sys.os],
            ['Kernel', sys.kernel],
            ['Arch', sys.arch],
            ['Host', sys.hostname],
            ['CPU', sys.cpu_model + ' (' + sys.cores + 'C)'],
            ['Uptime', sys.uptime_human]
        ];
        var html = '';
        for (var i = 0; i < rows.length; i++) {
            html += '<tr><td>' + rows[i][0] + '</td><td>' + rows[i][1] + '</td></tr>';
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
            '<span class="stat-value" style="color:' + thresholdColor(snap.cpu.max_percent) + '">' + snap.cpu.max_percent.toFixed(1) + '%</span></span>' +
            '<span class="color-muted" style="margin-left:12px">' + Math.round(snap.cpu.frequency_mhz) + 'MHz</span>';

        if (snap.cpu.per_core) {
            renderCores('cores-bar', snap.cpu.per_core);
        }
    }

    // --- Network Panel ---
    function renderNetwork(net, load) {
        dom.networkSummary.innerHTML =
            '<span class="net-stat"><span class="net-label">Load</span> ' +
            load.load1.toFixed(2) + ' ' + load.load5.toFixed(2) + ' ' + load.load15.toFixed(2) + '</span>' +
            '<span class="net-stat"><span class="net-down">&darr; ' + formatBytesPerSec(net.total_download_sec) + '</span></span>' +
            '<span class="net-stat"><span class="net-up">&uarr; ' + formatBytesPerSec(net.total_upload_sec) + '</span></span>' +
            '<span class="net-stat"><span class="net-label">TCP/UDP</span> ' + net.tcp_connections + ' / ' + net.udp_connections + '</span>';

        var ifaceHtml = '';
        for (var i = 0; i < net.interfaces.length; i++) {
            var iface = net.interfaces[i];
            var isIdle = iface.download_sec === 0 && iface.upload_sec === 0;
            var cls = isIdle ? ' color-muted' : '';
            ifaceHtml += '<div class="net-iface' + cls + '">' +
                '<span class="net-iface-name">' + iface.name + '</span>' +
                '<span class="net-down">&darr; ' + formatBytesPerSec(iface.download_sec) + '</span>' +
                '<span class="net-up" style="margin-left:8px">&uarr; ' + formatBytesPerSec(iface.upload_sec) + '</span>' +
                '</div>';
        }
        dom.networkInterfaces.innerHTML = ifaceHtml;
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

    // Initial REST fetch
    fetch('/api/v1/snapshot')
        .then(function(r) { return r.json(); })
        .then(function(data) { handleSnapshot(data); })
        .catch(function() { console.log('Initial fetch failed, waiting for WS'); });
})();
