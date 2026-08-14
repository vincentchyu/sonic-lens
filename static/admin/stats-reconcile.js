// static/admin/stats-reconcile.js
// Web 管理后台：播放统计一致性与数据对账控制脚本

async function triggerPlayStatsReconcile() {
    const btn = document.getElementById("btnRunReconcile");
    const statusText = document.getElementById("reconcileStatusText");

    if (!btn || !statusText) return;

    btn.disabled = true;
    btn.innerHTML = "⏳ 对账校对中...";
    statusText.style.color = "#ecc94b";
    statusText.innerText = "状态：正在执行全量流水与物理列对账...";

    try {
        const response = await fetch("/api/admin/stats/reconcile", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            }
        });

        const data = await response.json();
        if (response.ok && data.status === "success") {
            statusText.style.color = "#48bb78";
            statusText.innerText = "状态：对账校对完成 (" + new Date().toLocaleTimeString() + ")";
            if (typeof showToast === "function") {
                showToast("播放数据全量一致性对账与修复成功", "success");
            } else {
                alert("播放数据全量一致性对账与修复成功！");
            }
            // 刷新仪表盘
            if (typeof loadDashboardData === "function") {
                loadDashboardData();
            }
        } else {
            statusText.style.color = "#f56565";
            statusText.innerText = "状态：对账失败 (" + (data.error || "未知错误") + ")";
            if (typeof showToast === "function") {
                showToast("对账失败: " + (data.error || "未知错误"), "error");
            }
        }
    } catch (err) {
        statusText.style.color = "#f56565";
        statusText.innerText = "状态：请求异常 (" + err.message + ")";
    } finally {
        btn.disabled = false;
        btn.innerHTML = "🔄 播放量对账";
    }
}


