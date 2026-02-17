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
