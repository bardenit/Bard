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
});
