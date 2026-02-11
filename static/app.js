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
});
