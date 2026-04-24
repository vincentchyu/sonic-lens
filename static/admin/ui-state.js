(function () {
    function resolveTarget(target) {
        if (!target) return null;
        return typeof target === "string" ? document.getElementById(target) : target;
    }

    function escapeStateText(value) {
        return String(value || "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    function renderAdminState(target, className, message) {
        const element = resolveTarget(target);
        if (!element) return;
        element.innerHTML = `<div class="${className}">${escapeStateText(message)}</div>`;
    }

    window.renderAdminLoading = function (target, message = "加载中...") {
        renderAdminState(target, "loading", message);
    };

    window.renderAdminEmpty = function (target, message = "暂无数据") {
        renderAdminState(target, "no-data", message);
    };

    window.renderAdminError = function (target, error, prefix = "加载失败") {
        const message = error && error.message ? error.message : String(error || "未知错误");
        renderAdminState(target, "error", `${prefix}: ${message}`);
    };
})();
