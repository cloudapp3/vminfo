// Theme switcher for the vminfo dashboard.
// Themes are applied via [data-theme] on <html> (see dashboard.css). This module:
//   - exposes VMinfoTheme.colors (palette read from CSS vars) for other scripts
//   - wires the header dropdown, persists the choice, and follows the OS in "auto"
(function() {
    'use strict';

    var STORAGE_KEY = 'vminfo-theme';
    var PALETTE = ['green', 'cyan', 'blue', 'purple', 'pink', 'yellow', 'orange', 'red'];

    var colors = {};
    var listeners = [];

    function readColors() {
        var cs = getComputedStyle(document.documentElement);
        for (var i = 0; i < PALETTE.length; i++) {
            var name = PALETTE[i];
            var val = cs.getPropertyValue('--' + name).trim();
            colors[name] = val || colors[name] || '';
        }
    }

    function emit() {
        // Defer one frame so CSS variables from the new [data-theme] are applied
        // before we read them back and notify listeners.
        requestAnimationFrame(function() {
            readColors();
            for (var i = 0; i < listeners.length; i++) {
                try { listeners[i](); } catch (e) { console.error('theme listener error', e); }
            }
        });
    }

    // Populate synchronously on load (inline head script has already set the theme).
    readColors();

    window.VMinfoTheme = {
        colors: colors,
        get current() { return document.documentElement.getAttribute('data-theme') || 'auto'; },
        set: function(name) {
            document.documentElement.setAttribute('data-theme', name);
            try { localStorage.setItem(STORAGE_KEY, name); } catch (e) {}
            var select = document.getElementById('theme-select');
            if (select) select.value = name;
            emit();
        },
        onChange: function(cb) {
            if (typeof cb === 'function') listeners.push(cb);
        }
    };

    function init() {
        var select = document.getElementById('theme-select');
        if (!select) return;

        var current = document.documentElement.getAttribute('data-theme') || 'auto';
        select.value = current;
        select.addEventListener('change', function() {
            window.VMinfoTheme.set(select.value);
        });

        // When in "auto", refresh derived colors if the OS preference changes.
        var mq = window.matchMedia ? window.matchMedia('(prefers-color-scheme: light)') : null;
        if (mq) {
            var handler = function() {
                if ((document.documentElement.getAttribute('data-theme') || 'auto') === 'auto') {
                    emit();
                }
            };
            if (mq.addEventListener) mq.addEventListener('change', handler);
            else if (mq.addListener) mq.addListener(handler);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
