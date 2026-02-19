// Register service worker from root path so its scope covers the whole app.
if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/sw.js");
}

// PWA install prompt — captured and shown as a banner when available.
var _installPrompt = null;
window.addEventListener("beforeinstallprompt", function (e) {
    e.preventDefault();
    _installPrompt = e;
    try { if (sessionStorage.getItem("pwa-dismissed")) { return; } } catch(e2) {}
    var banner = document.getElementById("pwa-install-banner");
    if (banner) {
        banner.style.display = "flex";
    }
});

// Hide banner once installed.
window.addEventListener("appinstalled", function () {
    var banner = document.getElementById("pwa-install-banner");
    if (banner) { banner.style.display = "none"; }
    _installPrompt = null;
});

function pwaInstall() {
    if (!_installPrompt) { return; }
    _installPrompt.prompt();
    _installPrompt.userChoice.then(function () { _installPrompt = null; });
}

function pwaDismiss() {
    var banner = document.getElementById("pwa-install-banner");
    if (banner) { banner.style.display = "none"; }
    try { sessionStorage.setItem("pwa-dismissed", "1"); } catch(e) {}
}

document.addEventListener("DOMContentLoaded", function () {
    // PWA install banner buttons
    var pwaInstallBtn = document.getElementById("pwa-install-btn");
    if (pwaInstallBtn) { pwaInstallBtn.addEventListener("click", pwaInstall); }
    var pwaDismissBtn = document.getElementById("pwa-dismiss-btn");
    if (pwaDismissBtn) { pwaDismissBtn.addEventListener("click", pwaDismiss); }

    // Month select — auto-submit the period filter form on change
    var monthSel = document.getElementById("month-select");
    if (monthSel) {
        monthSel.addEventListener("change", function () {
            var v = this.value.split("-");
            this.form.elements["year"].value = v[0];
            this.form.elements["month"].value = v[1];
            this.form.submit();
        });
    }

    // Delete confirmation
    document.querySelectorAll("form[data-confirm]").forEach(function (form) {
        form.addEventListener("submit", function (e) {
            if (!confirm(form.dataset.confirm)) {
                e.preventDefault();
            }
        });
    });

    // Auto-dismiss flash messages
    var flash = document.querySelector(".flash");
    if (flash) {
        setTimeout(function () {
            flash.style.transition = "opacity 0.3s";
            flash.style.opacity = "0";
            setTimeout(function () { flash.remove(); }, 300);
        }, 3000);
    }

    // Nav dropdown: click/tap to toggle (supports mobile where hover doesn't work).
    document.querySelectorAll(".nav-dropdown").forEach(function (dropdown) {
        var toggle = dropdown.querySelector(".nav-dropdown-toggle");
        if (!toggle) return;
        toggle.addEventListener("click", function (e) {
            e.stopPropagation();
            var isOpen = dropdown.classList.contains("open");
            // Close all dropdowns first.
            document.querySelectorAll(".nav-dropdown.open").forEach(function (d) {
                d.classList.remove("open");
            });
            if (!isOpen) {
                dropdown.classList.add("open");
            }
        });
    });

    // Close dropdown when clicking anywhere else.
    document.addEventListener("click", function () {
        document.querySelectorAll(".nav-dropdown.open").forEach(function (d) {
            d.classList.remove("open");
        });
    });

    // Progress bars: read data-percent, set width and over/warn classes via JS
    // (JS-set element.style is NOT governed by CSP style-src)
    document.querySelectorAll(".progress-fill[data-percent]").forEach(function (el) {
        var pct = parseFloat(el.getAttribute("data-percent")) || 0;
        var clamped = Math.min(pct, 100);
        el.style.width = clamped + "%";
        if (pct >= 100) {
            el.classList.add("over");
        } else if (pct >= 80) {
            el.classList.add("warn");
        }
    });

    // Budget depth indentation: read data-depth, set paddingLeft via JS
    document.querySelectorAll("td[data-depth]").forEach(function (el) {
        var depth = parseInt(el.getAttribute("data-depth"), 10) || 0;
        if (depth > 0) {
            el.style.paddingLeft = (depth * 1.5) + "rem";
        }
    });
});
