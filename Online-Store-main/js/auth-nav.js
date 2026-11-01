/**
 * 导航登录态：登录后把“登录”入口替换为“我的”；管理员额外显示“后台”入口。
 * 所有页面加载；请求失败时保留默认的登录链接（游客状态）。
 */
(function () {
    'use strict';

    const slot = document.getElementById('nav-account');
    if (!slot || !window.api) {
        return;
    }

    window.api.session().then((user) => {
        if (!user) {
            return;
        }
        const link = slot.querySelector('a');
        const isActive = Boolean(link && link.classList.contains('active'));
        const activeAttr = isActive ? ' class="active" aria-current="page"' : '';
        const adminLink = user.role === 'admin' ? '<a href="admin.html">后台</a>' : '';
        slot.innerHTML = `${adminLink}<a href="account.html"${activeAttr}>我的</a>`;
    });
})();
