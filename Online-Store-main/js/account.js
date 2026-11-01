/**
 * 账户页逻辑：加载当前用户与订单列表，支持退出登录。
 */
(function () {
    'use strict';

    const api = window.api;
    if (!api) {
        console.error('API 模块未正确加载。');
        return;
    }

    const userLabel = document.getElementById('account-user');
    const ordersList = document.getElementById('orders-list');
    const banner = document.getElementById('order-banner');
    const logoutButton = document.getElementById('logout-btn');

    function escapeHtml(text) {
        return String(text == null ? '' : text)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    function formatDateTime(value) {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value || '';
        }
        const pad = (number) => String(number).padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} `
            + `${pad(date.getHours())}:${pad(date.getMinutes())}`;
    }

    function statusText(status) {
        const map = { pending: '待发货', paid: '已支付', shipped: '已发货', done: '已完成' };
        return map[status] || status;
    }

    function orderMarkup(order) {
        const rows = (order.items || []).map((item) => `
            <div class="order-item">
                <img src="${escapeHtml(item.image)}" alt="${escapeHtml(item.title)}">
                <div class="order-item-info">
                    <h4>${escapeHtml(item.title)}</h4>
                    <p>${escapeHtml(item.unitPriceText)} × ${item.quantity}</p>
                </div>
                <span class="order-item-total">${api.formatMoney(item.unitPriceCents * item.quantity)}</span>
            </div>
        `).join('');

        return `
            <article class="order-card">
                <header class="order-head">
                    <span class="order-id">订单号 ${escapeHtml(order.id)}</span>
                    <span class="order-status">${escapeHtml(statusText(order.status))}</span>
                    <time>${escapeHtml(formatDateTime(order.createdAt))}</time>
                </header>
                <div class="order-items">${rows}</div>
                <div class="order-foot">
                    <span class="order-shipping">
                        收货：${escapeHtml(order.shippingName)} · ${escapeHtml(order.shippingPhone)}
                        · ${escapeHtml(order.shippingAddress)}
                    </span>
                    <span class="order-total">${api.formatMoney(order.totalCents)}</span>
                </div>
            </article>
        `;
    }

    async function load() {
        let user = null;
        try {
            const data = await api.get('/api/auth/me');
            user = data && data.user;
        } catch (error) {
            if (error.status === 401) {
                window.location.href = 'login.html?next=account.html';
                return;
            }
            // 网络故障等服务端不可达时不推去登录页，原地提示即可。
            ordersList.innerHTML = `<p class="orders-error">${api.escapeHtml(error.message)}</p>`;
            return;
        }

        userLabel.textContent = `${user.username}（${user.email}）`;

        const params = new URLSearchParams(window.location.search);
        const newOrder = params.get('order');
        if (newOrder) {
            banner.textContent = `订单 ${newOrder} 提交成功，我们会尽快安排发货。`;
            banner.hidden = false;
            history.replaceState(null, '', 'account.html');
        }

        try {
            const data = await api.get('/api/orders');
            const orders = data.orders || [];

            if (!orders.length) {
                ordersList.innerHTML = `
                    <div class="orders-empty">
                        <i class="fas fa-box-open" aria-hidden="true"></i>
                        <h3>还没有订单</h3>
                        <p>去挑选新鲜的有机好物，开启健康生活吧！</p>
                        <a href="products.html" class="shop-btn">去逛逛</a>
                    </div>
                `;
                return;
            }
            ordersList.innerHTML = orders.map(orderMarkup).join('');
        } catch (error) {
            ordersList.innerHTML = `<p class="orders-error">${escapeHtml(error.message)}</p>`;
        }
    }

    logoutButton.addEventListener('click', async () => {
        logoutButton.disabled = true;
        try {
            await api.post('/api/auth/logout');
        } catch (error) {
            // 即使接口失败也继续清理本地状态。
        }
        if (window.greenCart) {
            window.greenCart.clearLocal();
        } else {
            try {
                localStorage.removeItem('greenCart');
            } catch (error) {
                /* 忽略 */
            }
        }
        window.location.href = 'index.html';
    });

    load();
})();
