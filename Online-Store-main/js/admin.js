/**
 * 管理后台单页应用：hash 路由（概览/商品/用户/订单）。
 * 无构建、无模块，仅依赖 window.api（js/api.js）。
 */
(function () {
    'use strict';

    const api = window.api;
    if (!api) {
        document.getElementById('admin-fatal-message').textContent = 'API 模块未正确加载。';
        document.getElementById('admin-fatal').hidden = false;
        return;
    }

    const escapeHtml = api.escapeHtml;
    const formatMoney = api.formatMoney;

    const VIEWS = { dashboard: '概览', products: '商品管理', users: '用户管理', orders: '订单管理' };
    const STATUS_TEXT = {
        pending: '待发货', paid: '已支付', shipped: '已发货', done: '已完成', cancelled: '已取消'
    };
    const PAGE_SIZES = [10, 20, 50];

    const state = {
        user: null,
        categories: [],
        view: 'dashboard',
        statsDirty: true,
        products: { page: 1, pageSize: 20, query: '', category: '', active: 'all', seq: 0 },
        users: { page: 1, pageSize: 20, query: '', role: 'all', seq: 0 },
        orders: { page: 1, pageSize: 20, status: 'all', query: '', userId: 0, userName: '', seq: 0 }
    };

    const $ = (id) => document.getElementById(id);

    /* ---------- 基础工具 ---------- */

    // 统一请求：401/403 跳转登录页，其余错误交给调用方展示。
    async function request(path, options) {
        try {
            return await api.request(path, options);
        } catch (error) {
            if (error.status === 401) {
                window.location.replace('admin-login.html');
            } else if (error.status === 403) {
                window.location.replace('admin-login.html?error=not_admin');
            }
            throw error;
        }
    }

    let toastTimer = null;
    function toast(message, isError) {
        const el = $('admin-toast');
        el.textContent = message;
        el.classList.toggle('toast-error', Boolean(isError));
        el.hidden = false;
        clearTimeout(toastTimer);
        toastTimer = setTimeout(() => { el.hidden = true; }, 3000);
    }

    function showError(id, message) {
        const el = $(id);
        el.textContent = message;
        el.hidden = false;
    }

    function hideError(id) {
        $(id).hidden = true;
    }

    function formatDateTime(value) {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value || '-';
        }
        const pad = (number) => String(number).padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} `
            + `${pad(date.getHours())}:${pad(date.getMinutes())}`;
    }

    function statusBadge(status) {
        const text = STATUS_TEXT[status] || status;
        return `<span class="admin-badge badge-${escapeHtml(status)}">${escapeHtml(text)}</span>`;
    }

    function categoryTitle(slug) {
        const found = state.categories.find((category) => category.slug === slug);
        return found ? found.title : slug;
    }

    function thumb(image, alt) {
        if (image) {
            return `<img class="admin-thumb" src="${escapeHtml(image)}" alt="${escapeHtml(alt)}" loading="lazy">`;
        }
        return '<span class="admin-thumb-placeholder"><i class="fas fa-image" aria-hidden="true"></i></span>';
    }

    /* ---------- 弹窗 ---------- */

    function openModal(id) {
        const modal = $(id);
        modal.lastFocused = document.activeElement;
        modal.hidden = false;
        const first = modal.querySelector('input, select, textarea, button:not(.admin-modal-close)');
        if (first) {
            first.focus();
        }
    }

    function closeModal(modal) {
        modal.hidden = true;
        if (modal.lastFocused && typeof modal.lastFocused.focus === 'function') {
            modal.lastFocused.focus();
        }
    }

    document.addEventListener('keydown', (event) => {
        if (event.key !== 'Escape') {
            return;
        }
        const open = document.querySelector('.admin-modal:not([hidden])');
        if (open) {
            closeModal(open);
        }
    });

    let confirmHandler = null;
    function confirmAction(message, handler) {
        $('confirm-message').textContent = message;
        confirmHandler = handler;
        openModal('confirm-modal');
    }

    /* ---------- 概览 ---------- */

    let statsSeq = 0;
    async function loadStats() {
        const seq = ++statsSeq;
        hideError('stats-error');
        try {
            const data = await request('/api/admin/stats');
            if (seq !== statsSeq || state.view !== 'dashboard') {
                return;
            }
            state.statsDirty = false;
            renderStats(data.stats || {}, data.recentOrders || []);
        } catch (error) {
            if (seq !== statsSeq || error.status === 401 || error.status === 403) {
                return;
            }
            showError('stats-error', error.message);
        }
    }

    function renderStats(stats, recentOrders) {
        $('stat-users').textContent = stats.userCount ?? '-';
        $('stat-admins').textContent = `含管理员 ${stats.adminCount ?? 0} 名`;
        $('stat-products').textContent = stats.productCount ?? '-';
        $('stat-active-products').textContent = `上架 ${stats.activeProductCount ?? 0} 个`;
        $('stat-orders').textContent = stats.orderCount ?? '-';
        $('stat-pending-orders').textContent = `待发货 ${stats.pendingOrderCount ?? 0} 笔`;
        $('stat-revenue').textContent = formatMoney(stats.revenueCents ?? 0);
        $('stat-today-orders').textContent = `今日 ${stats.todayOrderCount ?? 0} 笔`;
        $('stat-grid').hidden = false;

        const body = $('recent-orders-body');
        if (!recentOrders.length) {
            body.innerHTML = '<tr><td colspan="7" class="admin-empty">还没有订单</td></tr>';
            return;
        }
        body.innerHTML = recentOrders.map((order) => `
            <tr>
                <td>${escapeHtml(order.id)}</td>
                <td>${escapeHtml(order.username)}<span class="cell-sub">${escapeHtml(order.userEmail)}</span></td>
                <td>${order.itemCount}</td>
                <td>${formatMoney(order.totalCents)}</td>
                <td>${statusBadge(order.status)}</td>
                <td>${escapeHtml(formatDateTime(order.createdAt))}</td>
                <td><button type="button" class="admin-btn admin-btn-sm"
                    data-action="order-view" data-id="${escapeHtml(order.id)}">查看</button></td>
            </tr>
        `).join('');
    }

    /* ---------- 分页 ---------- */

    function renderPagination(container, list, total, reload) {
        const pages = Math.max(1, Math.ceil(total / list.pageSize));
        const sizeOptions = PAGE_SIZES.map((size) => `
            <option value="${size}"${size === list.pageSize ? ' selected' : ''}>${size} 条/页</option>
        `).join('');
        container.innerHTML = `
            <span>第 ${list.page} / ${pages} 页 · 共 ${total} 条</span>
            <select class="admin-input" data-page-size aria-label="每页条数">${sizeOptions}</select>
            <button type="button" class="admin-btn admin-btn-sm" data-page-prev
                ${list.page <= 1 ? 'disabled' : ''}>上一页</button>
            <button type="button" class="admin-btn admin-btn-sm" data-page-next
                ${list.page >= pages ? 'disabled' : ''}>下一页</button>
        `;
        container.querySelector('[data-page-prev]').addEventListener('click', () => {
            if (list.page > 1) {
                list.page -= 1;
                reload();
            }
        });
        container.querySelector('[data-page-next]').addEventListener('click', () => {
            const maxPage = Math.max(1, Math.ceil(total / list.pageSize));
            if (list.page < maxPage) {
                list.page += 1;
                reload();
            }
        });
        container.querySelector('[data-page-size]').addEventListener('change', (event) => {
            list.pageSize = Number(event.target.value) || 20;
            list.page = 1;
            reload();
        });
    }

    // 删除后若当前页只剩这一条且不在第一页，则回退一页再加载。
    function stepBackIfLastItem(list, reload) {
        if (list.lastCount === 1 && list.page > 1) {
            list.page -= 1;
        }
        reload();
    }

    /* ---------- 商品管理 ---------- */

    let categoriesLoaded = false;
    async function loadCategories() {
        if (categoriesLoaded) {
            return;
        }
        try {
            const data = await request('/api/categories');
            state.categories = data.categories || [];
            categoriesLoaded = true;
            const filter = $('product-category');
            filter.innerHTML = '<option value="">全部分类</option>' + state.categories.map((category) => `
                <option value="${escapeHtml(category.slug)}">${escapeHtml(category.title)}</option>
            `).join('');
            const formSelect = $('pf-category');
            formSelect.innerHTML = '<option value="">请选择</option>' + state.categories.map((category) => `
                <option value="${escapeHtml(category.slug)}">${escapeHtml(category.title)}</option>
            `).join('');
        } catch (error) {
            toast(error.message, true);
        }
    }

    async function loadProducts() {
        const list = state.products;
        const seq = ++list.seq;
        hideError('products-error');
        const params = new URLSearchParams({
            page: String(list.page), pageSize: String(list.pageSize)
        });
        if (list.query) {
            params.set('query', list.query);
        }
        if (list.category) {
            params.set('category', list.category);
        }
        if (list.active !== 'all') {
            params.set('active', list.active);
        }
        try {
            const data = await request('/api/admin/products?' + params.toString());
            if (seq !== list.seq || state.view !== 'products') {
                return;
            }
            renderProducts(data.products || [], data.total || 0);
        } catch (error) {
            if (seq !== list.seq || error.status === 401 || error.status === 403) {
                return;
            }
            showError('products-error', error.message);
        }
    }

    function renderProducts(products, total) {
        const body = $('products-body');
        if (!products.length) {
            body.innerHTML = '<tr><td colspan="7" class="admin-empty">没有符合条件的商品</td></tr>';
        } else {
            body.innerHTML = products.map((product) => `
                <tr>
                    <td>${thumb(product.image, product.title)}</td>
                    <td><strong>${escapeHtml(product.title)}</strong>
                        <span class="cell-sub cell-id">${escapeHtml(product.id)}</span></td>
                    <td>${escapeHtml(categoryTitle(product.category))}</td>
                    <td>${escapeHtml(product.priceText)}</td>
                    <td>${product.rating > 0 ? escapeHtml(product.rating) : '-'}</td>
                    <td><span class="admin-badge ${product.active ? 'badge-active' : 'badge-inactive'}">
                        ${product.active ? '已上架' : '已下架'}</span></td>
                    <td class="admin-actions">
                        <button type="button" class="admin-btn admin-btn-sm"
                            data-action="product-edit" data-id="${escapeHtml(product.id)}">编辑</button>
                        <button type="button" class="admin-btn admin-btn-sm"
                            data-action="product-toggle" data-id="${escapeHtml(product.id)}"
                            data-active="${product.active}">${product.active ? '下架' : '上架'}</button>
                        <button type="button" class="admin-btn admin-btn-sm admin-btn-link-danger"
                            data-action="product-delete" data-id="${escapeHtml(product.id)}"
                            data-title="${escapeHtml(product.title)}">删除</button>
                    </td>
                </tr>
            `).join('');
        }
        state.products.lastCount = products.length;
        renderPagination($('products-pagination'), state.products, total, loadProducts);
    }

    function openProductModal(product) {
        const form = $('product-form');
        form.reset();
        $('pf-error').hidden = true;
        $('pf-image-preview').hidden = true;
        if (product) {
            $('product-modal-title').textContent = '编辑商品';
            form.dataset.editId = product.id;
            $('pf-title').value = product.title;
            $('pf-category').value = product.category;
            $('pf-type').value = product.type;
            $('pf-price').value = (product.priceCents / 100).toFixed(2);
            $('pf-unit').value = product.unit;
            $('pf-rating').value = product.rating > 0 ? product.rating : '';
            $('pf-sort').value = product.sortOrder ?? '';
            $('pf-image').value = product.image;
            $('pf-active').value = product.active ? 'true' : 'false';
            $('pf-description').value = product.description || '';
            $('pf-detail').value = product.detail || '';
            $('pf-origin').value = product.origin || '';
            $('pf-storage').value = product.storage || '';
            $('pf-nutrients').value = product.nutrients || '';
            $('pf-care').value = product.care || '';
            $('pf-color').value = product.color || '';
            updateImagePreview();
        } else {
            $('product-modal-title').textContent = '新增商品';
            delete form.dataset.editId;
            $('pf-active').value = 'true';
        }
        openModal('product-modal');
    }

    function updateImagePreview() {
        const value = $('pf-image').value.trim();
        const preview = $('pf-image-preview');
        if (!value || value.includes('://') || value.includes('..')) {
            preview.hidden = true;
            preview.removeAttribute('src');
            return;
        }
        preview.onerror = () => {
            preview.hidden = true;
        };
        preview.onload = () => {
            preview.hidden = false;
        };
        preview.src = value;
    }

    async function submitProductForm(event) {
        event.preventDefault();
        const form = $('product-form');
        const errorEl = $('pf-error');
        errorEl.hidden = true;

        const priceYuan = Number($('pf-price').value);
        if (!Number.isFinite(priceYuan) || priceYuan < 0) {
            errorEl.textContent = '请输入有效的价格。';
            errorEl.hidden = false;
            return;
        }

        const payload = {
            title: $('pf-title').value.trim(),
            category: $('pf-category').value,
            type: $('pf-type').value,
            priceCents: Math.round(priceYuan * 100),
            unit: $('pf-unit').value.trim(),
            image: $('pf-image').value.trim(),
            active: $('pf-active').value === 'true',
            description: $('pf-description').value.trim(),
            detail: $('pf-detail').value.trim(),
            origin: $('pf-origin').value.trim(),
            storage: $('pf-storage').value.trim(),
            nutrients: $('pf-nutrients').value.trim(),
            care: $('pf-care').value.trim(),
            color: $('pf-color').value.trim()
        };
        if ($('pf-rating').value !== '') {
            payload.rating = Number($('pf-rating').value);
        }
        if ($('pf-sort').value !== '') {
            payload.sortOrder = Number($('pf-sort').value);
        }

        const editId = form.dataset.editId;
        const submitButton = $('pf-submit');
        submitButton.disabled = true;
        try {
            if (editId) {
                await request('/api/admin/products/' + encodeURIComponent(editId), {
                    method: 'PATCH', body: payload
                });
                toast('商品已保存');
            } else {
                await request('/api/admin/products', { method: 'POST', body: payload });
                toast('商品已创建');
            }
            state.statsDirty = true;
            closeModal($('product-modal'));
            loadProducts();
        } catch (error) {
            if (error.status !== 401 && error.status !== 403) {
                errorEl.textContent = error.message;
                errorEl.hidden = false;
            }
        } finally {
            submitButton.disabled = false;
        }
    }

    /* ---------- 用户管理 ---------- */

    async function loadUsers() {
        const list = state.users;
        const seq = ++list.seq;
        hideError('users-error');
        const params = new URLSearchParams({
            page: String(list.page), pageSize: String(list.pageSize)
        });
        if (list.query) {
            params.set('query', list.query);
        }
        if (list.role !== 'all') {
            params.set('role', list.role);
        }
        try {
            const data = await request('/api/admin/users?' + params.toString());
            if (seq !== list.seq || state.view !== 'users') {
                return;
            }
            renderUsers(data.users || [], data.total || 0);
        } catch (error) {
            if (seq !== list.seq || error.status === 401 || error.status === 403) {
                return;
            }
            showError('users-error', error.message);
        }
    }

    function renderUsers(users, total) {
        const body = $('users-body');
        const selfId = state.user ? String(state.user.id) : '';
        if (!users.length) {
            body.innerHTML = '<tr><td colspan="8" class="admin-empty">没有符合条件的用户</td></tr>';
        } else {
            body.innerHTML = users.map((user) => {
                const isSelf = String(user.id) === selfId;
                return `
                <tr>
                    <td>${user.id}</td>
                    <td>${escapeHtml(user.username)}</td>
                    <td>${escapeHtml(user.email)}</td>
                    <td><span class="admin-badge ${user.role === 'admin' ? 'badge-admin' : 'badge-user'}">
                        ${user.role === 'admin' ? '管理员' : '普通用户'}</span></td>
                    <td>${user.orderCount}</td>
                    <td>${formatMoney(user.totalSpentCents)}</td>
                    <td>${escapeHtml(formatDateTime(user.createdAt))}</td>
                    <td class="admin-actions">
                        <button type="button" class="admin-btn admin-btn-sm" data-action="user-role"
                            data-id="${user.id}" data-role="${escapeHtml(user.role)}"
                            ${isSelf ? 'disabled title="不能修改自己的角色"' : ''}>
                            ${user.role === 'admin' ? '取消管理员' : '设为管理员'}</button>
                        <button type="button" class="admin-btn admin-btn-sm" data-action="user-password"
                            data-id="${user.id}" data-name="${escapeHtml(user.username)}">重置密码</button>
                        <button type="button" class="admin-btn admin-btn-sm" data-action="user-orders"
                            data-id="${user.id}" data-name="${escapeHtml(user.username)}">查看订单</button>
                        <button type="button" class="admin-btn admin-btn-sm admin-btn-link-danger"
                            data-action="user-delete" data-id="${user.id}" data-name="${escapeHtml(user.username)}"
                            ${isSelf ? 'disabled title="不能删除自己"' : ''}>删除</button>
                    </td>
                </tr>
                `;
            }).join('');
        }
        state.users.lastCount = users.length;
        renderPagination($('users-pagination'), state.users, total, loadUsers);
    }

    /* ---------- 订单管理 ---------- */

    async function loadOrders() {
        const list = state.orders;
        const seq = ++list.seq;
        hideError('orders-error');
        const params = new URLSearchParams({
            page: String(list.page), pageSize: String(list.pageSize)
        });
        if (list.status !== 'all') {
            params.set('status', list.status);
        }
        if (list.query) {
            params.set('query', list.query);
        }
        if (list.userId > 0) {
            params.set('userId', String(list.userId));
        }
        const chip = $('order-user-filter');
        chip.hidden = list.userId <= 0;
        chip.textContent = `仅看用户：${list.userName || list.userId}（清除）`;

        try {
            const data = await request('/api/admin/orders?' + params.toString());
            if (seq !== list.seq || state.view !== 'orders') {
                return;
            }
            renderOrders(data.orders || [], data.total || 0);
        } catch (error) {
            if (seq !== list.seq || error.status === 401 || error.status === 403) {
                return;
            }
            showError('orders-error', error.message);
        }
    }

    function renderOrders(orders, total) {
        const body = $('orders-body');
        if (!orders.length) {
            body.innerHTML = '<tr><td colspan="8" class="admin-empty">没有符合条件的订单</td></tr>';
        } else {
            body.innerHTML = orders.map((order) => `
                <tr>
                    <td>${escapeHtml(order.id)}</td>
                    <td>${escapeHtml(order.username)}<span class="cell-sub">${escapeHtml(order.userEmail)}</span></td>
                    <td>${escapeHtml(order.shippingName)}<span class="cell-sub">${escapeHtml(order.shippingPhone)}</span></td>
                    <td>${order.itemCount}</td>
                    <td>${formatMoney(order.totalCents)}</td>
                    <td>
                        <label class="visually-hidden" for="order-status-${escapeHtml(order.id)}">订单状态</label>
                        <select class="admin-input" id="order-status-${escapeHtml(order.id)}"
                            data-action="order-status" data-id="${escapeHtml(order.id)}"
                            data-previous="${escapeHtml(order.status)}">
                            ${Object.keys(STATUS_TEXT).map((status) => `
                                <option value="${status}"${status === order.status ? ' selected' : ''}>
                                    ${STATUS_TEXT[status]}</option>
                            `).join('')}
                        </select>
                    </td>
                    <td>${escapeHtml(formatDateTime(order.createdAt))}</td>
                    <td class="admin-actions">
                        <button type="button" class="admin-btn admin-btn-sm"
                            data-action="order-view" data-id="${escapeHtml(order.id)}">详情</button>
                        <button type="button" class="admin-btn admin-btn-sm admin-btn-link-danger"
                            data-action="order-delete" data-id="${escapeHtml(order.id)}">删除</button>
                    </td>
                </tr>
            `).join('');
        }
        state.orders.lastCount = orders.length;
        renderPagination($('orders-pagination'), state.orders, total, loadOrders);
    }

    async function openOrderModal(orderId) {
        try {
            const data = await request('/api/admin/orders/' + encodeURIComponent(orderId));
            const order = data.order;
            $('order-modal-title').textContent = `订单 ${order.id}`;
            const items = (order.items || []).map((item) => `
                <tr>
                    <td>${thumb(item.image, item.title)}</td>
                    <td>${escapeHtml(item.title)}</td>
                    <td>${escapeHtml(item.unitPriceText)} × ${item.quantity}</td>
                    <td>${formatMoney(item.unitPriceCents * item.quantity)}</td>
                </tr>
            `).join('');
            $('order-detail').innerHTML = `
                <dl class="order-detail-meta">
                    <div><dt>买家</dt><dd>${escapeHtml(order.username)}（${escapeHtml(order.userEmail)}）</dd></div>
                    <div><dt>状态</dt><dd>${statusBadge(order.status)}</dd></div>
                    <div><dt>下单时间</dt><dd>${escapeHtml(formatDateTime(order.createdAt))}</dd></div>
                    <div><dt>收货人</dt><dd>${escapeHtml(order.shippingName)}</dd></div>
                    <div><dt>联系电话</dt><dd>${escapeHtml(order.shippingPhone)}</dd></div>
                    <div><dt>收货地址</dt><dd>${escapeHtml(order.shippingAddress)}</dd></div>
                </dl>
                <table class="order-detail-items" aria-label="订单商品明细">
                    <thead><tr><th scope="col">图片</th><th scope="col">商品</th>
                        <th scope="col">单价 × 数量</th><th scope="col">小计</th></tr></thead>
                    <tbody>${items}</tbody>
                </table>
                <p class="order-detail-total">订单总额：${formatMoney(order.totalCents)}</p>
            `;
            openModal('order-modal');
        } catch (error) {
            if (error.status !== 401 && error.status !== 403) {
                toast(error.message, true);
            }
        }
    }

    /* ---------- 事件绑定 ---------- */

    function bindEvents() {
        $('stats-refresh').addEventListener('click', loadStats);

        $('product-refresh').addEventListener('click', loadProducts);
        $('product-add').addEventListener('click', () => openProductModal(null));
        $('product-active').addEventListener('change', (event) => {
            state.products.active = event.target.value;
            state.products.page = 1;
            loadProducts();
        });
        $('product-category').addEventListener('change', (event) => {
            state.products.category = event.target.value;
            state.products.page = 1;
            loadProducts();
        });
        $('pf-image').addEventListener('input', updateImagePreview);
        $('product-form').addEventListener('submit', submitProductForm);

        $('user-refresh').addEventListener('click', loadUsers);
        $('user-role').addEventListener('change', (event) => {
            state.users.role = event.target.value;
            state.users.page = 1;
            loadUsers();
        });

        $('order-refresh').addEventListener('click', loadOrders);
        $('order-status').addEventListener('change', (event) => {
            state.orders.status = event.target.value;
            state.orders.page = 1;
            loadOrders();
        });
        $('order-user-filter').addEventListener('click', () => {
            state.orders.userId = 0;
            state.orders.userName = '';
            state.orders.page = 1;
            loadOrders();
        });

        // 搜索防抖（300ms）
        const debounce = (fn) => {
            let timer = null;
            return (event) => {
                clearTimeout(timer);
                timer = setTimeout(fn, 300);
            };
        };
        $('product-search').addEventListener('input', debounce(() => {
            state.products.query = $('product-search').value.trim();
            state.products.page = 1;
            loadProducts();
        }));
        $('user-search').addEventListener('input', debounce(() => {
            state.users.query = $('user-search').value.trim();
            state.users.page = 1;
            loadUsers();
        }));
        $('order-search').addEventListener('input', debounce(() => {
            state.orders.query = $('order-search').value.trim();
            state.orders.page = 1;
            loadOrders();
        }));

        // 表格操作（事件委托）
        $('products-body').addEventListener('click', async (event) => {
            const button = event.target.closest('button[data-action]');
            if (!button) {
                return;
            }
            const id = button.dataset.id;
            if (button.dataset.action === 'product-edit') {
                try {
                    const data = await request('/api/admin/products/' + encodeURIComponent(id));
                    openProductModal(data.product);
                } catch (error) {
                    if (error.status !== 401 && error.status !== 403) {
                        toast(error.message, true);
                    }
                }
            } else if (button.dataset.action === 'product-toggle') {
                const active = button.dataset.active === 'true';
                button.disabled = true;
                try {
                    await request('/api/admin/products/' + encodeURIComponent(id), {
                        method: 'PATCH', body: { active: !active }
                    });
                    toast(active ? '商品已下架' : '商品已上架');
                    state.statsDirty = true;
                    loadProducts();
                } catch (error) {
                    button.disabled = false;
                    if (error.status !== 401 && error.status !== 403) {
                        toast(error.message, true);
                    }
                }
            } else if (button.dataset.action === 'product-delete') {
                confirmAction(
                    `确定删除商品「${button.dataset.title}」吗？商品会从商城与购物车移除，历史订单快照保留。`,
                    async () => {
                        await request('/api/admin/products/' + encodeURIComponent(id), { method: 'DELETE' });
                        toast('商品已删除');
                        state.statsDirty = true;
                        stepBackIfLastItem(state.products, loadProducts);
                    }
                );
            }
        });

        $('users-body').addEventListener('click', async (event) => {
            const button = event.target.closest('button[data-action]');
            if (!button || button.disabled) {
                return;
            }
            const id = button.dataset.id;
            if (button.dataset.action === 'user-role') {
                const promoting = button.dataset.role !== 'admin';
                confirmAction(
                    promoting ? '确定将该用户设为管理员吗？'
                        : '确定取消该用户的管理员角色吗？',
                    async () => {
                        await request(`/api/admin/users/${encodeURIComponent(id)}`, {
                            method: 'PATCH', body: { role: promoting ? 'admin' : 'user' }
                        });
                        toast(promoting ? '已设为管理员' : '已取消管理员');
                        state.statsDirty = true;
                        loadUsers();
                    }
                );
            } else if (button.dataset.action === 'user-password') {
                const form = $('password-form');
                form.reset();
                $('pm-error').hidden = true;
                form.dataset.userId = id;
                $('password-modal-title').textContent = `重置密码（${button.dataset.name}）`;
                openModal('password-modal');
            } else if (button.dataset.action === 'user-orders') {
                state.orders.userId = Number(id);
                state.orders.userName = button.dataset.name;
                state.orders.page = 1;
                state.orders.status = 'all';
                $('order-status').value = 'all';
                if (state.view === 'orders') {
                    loadOrders();
                } else {
                    window.location.hash = '#orders';
                }
            } else if (button.dataset.action === 'user-delete') {
                confirmAction(
                    `确定删除用户「${button.dataset.name}」吗？该用户的会话、购物车与订单会被级联删除。`,
                    async () => {
                        await request(`/api/admin/users/${encodeURIComponent(id)}`, { method: 'DELETE' });
                        toast('用户已删除');
                        state.statsDirty = true;
                        stepBackIfLastItem(state.users, loadUsers);
                    }
                );
            }
        });

        // 行内状态修改
        $('orders-body').addEventListener('change', async (event) => {
            const select = event.target.closest('select[data-action="order-status"]');
            if (!select) {
                return;
            }
            const previous = select.dataset.previous || select.value;
            select.disabled = true;
            try {
                await request('/api/admin/orders/' + encodeURIComponent(select.dataset.id), {
                    method: 'PATCH', body: { status: select.value }
                });
                toast('订单状态已更新');
                state.statsDirty = true;
                loadOrders();
            } catch (error) {
                select.value = previous;
                if (error.status !== 401 && error.status !== 403) {
                    toast(error.message, true);
                }
            } finally {
                select.disabled = false;
            }
        });

        $('orders-body').addEventListener('click', async (event) => {
            const button = event.target.closest('button[data-action]');
            if (!button) {
                return;
            }
            if (button.dataset.action === 'order-view') {
                openOrderModal(button.dataset.id);
            } else if (button.dataset.action === 'order-delete') {
                confirmAction(
                    `确定删除订单 ${button.dataset.id} 吗？订单明细会一并删除。`,
                    async () => {
                        await request('/api/admin/orders/' + encodeURIComponent(button.dataset.id),
                            { method: 'DELETE' });
                        toast('订单已删除');
                        state.statsDirty = true;
                        stepBackIfLastItem(state.orders, loadOrders);
                    }
                );
            }
        });

        // 确认弹窗
        $('confirm-ok').addEventListener('click', async () => {
            if (!confirmHandler) {
                closeModal($('confirm-modal'));
                return;
            }
            const handler = confirmHandler;
            confirmHandler = null;
            const button = $('confirm-ok');
            button.disabled = true;
            try {
                await handler();
            } catch (error) {
                if (error.status !== 401 && error.status !== 403) {
                    toast(error.message, true);
                }
            } finally {
                button.disabled = false;
                closeModal($('confirm-modal'));
            }
        });

        // 重置密码表单
        $('password-form').addEventListener('submit', async (event) => {
            event.preventDefault();
            const form = event.target;
            const errorEl = $('pm-error');
            errorEl.hidden = true;
            const password = $('pm-password').value;
            if (password.length < 8 || password.length > 72) {
                errorEl.textContent = '密码长度需要 8-72 个字符。';
                errorEl.hidden = false;
                return;
            }
            if (password !== $('pm-confirm').value) {
                errorEl.textContent = '两次输入的密码不一致。';
                errorEl.hidden = false;
                return;
            }
            const userId = form.dataset.userId;
            const submitButton = $('pm-submit');
            submitButton.disabled = true;
            try {
                await request(`/api/admin/users/${encodeURIComponent(userId)}/password`, {
                    method: 'POST', body: { password }
                });
                toast('密码已重置，该用户全部会话已注销');
                closeModal($('password-modal'));
                if (state.user && String(state.user.id) === String(userId)) {
                    // 重置自己的密码：会话已失效，回到登录页。
                    window.location.replace('admin-login.html');
                }
            } catch (error) {
                if (error.status !== 401 && error.status !== 403) {
                    errorEl.textContent = error.message;
                    errorEl.hidden = false;
                }
            } finally {
                submitButton.disabled = false;
            }
        });

        // 通用关闭（遮罩 / 取消 / ×）
        document.querySelectorAll('.admin-modal').forEach((modal) => {
            modal.addEventListener('click', (event) => {
                if (event.target.closest('[data-close-modal]')) {
                    if (modal.id === 'confirm-modal') {
                        confirmHandler = null;
                    }
                    closeModal(modal);
                }
            });
        });

        // 退出登录
        $('logout-btn').addEventListener('click', async () => {
            const button = $('logout-btn');
            button.disabled = true;
            try {
                await api.post('/api/auth/logout');
            } catch (error) {
                // 无论接口结果如何都清理本地状态并返回登录页。
            }
            window.location.replace('admin-login.html');
        });

        // 全页错误重试
        $('admin-retry').addEventListener('click', () => {
            window.location.reload();
        });

        // 路由
        window.addEventListener('hashchange', () => {
            switchView(currentViewFromHash());
        });
    }

    /* ---------- 路由与启动 ---------- */

    function currentViewFromHash() {
        const hash = (window.location.hash || '').replace('#', '');
        return VIEWS[hash] ? hash : 'dashboard';
    }

    function switchView(view) {
        state.view = view;
        document.querySelectorAll('.admin-view').forEach((section) => {
            section.hidden = section.id !== 'view-' + view;
        });
        document.querySelectorAll('.admin-nav-link').forEach((link) => {
            link.classList.toggle('active', link.dataset.view === view);
        });
        $('view-title').textContent = VIEWS[view];
        if (view === 'dashboard') {
            if (state.statsDirty) {
                loadStats();
            }
        } else if (view === 'products') {
            loadCategories().then(loadProducts);
        } else if (view === 'users') {
            loadUsers();
        } else if (view === 'orders') {
            loadOrders();
        }
    }

    function showFatal(message) {
        $('admin-fatal-message').textContent = message;
        $('admin-fatal').hidden = false;
    }

    async function init() {
        try {
            const data = await api.request('/api/admin/auth/me');
            state.user = data.user;
        } catch (error) {
            if (error.status === 401 || error.status === 403) {
                window.location.replace(
                    error.status === 403 ? 'admin-login.html?error=not_admin' : 'admin-login.html'
                );
                return;
            }
            showFatal(error.message || '无法连接服务器，请确认后端服务已启动。');
            return;
        }

        $('admin-who').textContent = `${state.user.username}（${state.user.email}）`;
        bindEvents();
        $('admin-app').hidden = false;
        switchView(currentViewFromHash());
    }

    init();
})();
