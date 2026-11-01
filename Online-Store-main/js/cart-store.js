/**
 * 购物车数据层（双模式）。
 * - 游客：localStorage（键 greenCart），保存商品快照；
 * - 登录后：服务端 /api/cart，价格以服务端商品表为准。
 * 页面通过 onChange 订阅变化；所有修改方法返回 Promise。
 */
window.greenCart = (function () {
    'use strict';

    const api = window.api;
    if (!api) {
        console.error('购物车模块依赖 js/api.js，请先加载。');
        return null;
    }

    const STORAGE_KEY = 'greenCart';
    const MIN_QUANTITY = 1;
    const MAX_QUANTITY = 99;

    let mode = 'guest';
    let items = [];
    const listeners = new Set();

    function parsePrice(priceText) {
        const match = String(priceText || '').match(/[\d.]+/);
        return match ? parseFloat(match[0]) : 0;
    }

    function normalizeImagePath(path) {
        return String(path || '').replace(/^(\.\.\/)+/, '');
    }

    function normalizeItem(raw) {
        const priceStr = raw.priceStr || raw.price || '';
        const priceNum = Number(raw.priceNum);
        const quantity = parseInt(raw.quantity, 10);

        return {
            id: String(raw.id || ''),
            title: String(raw.title || ''),
            priceStr,
            priceNum: Number.isFinite(priceNum) && priceNum > 0 ? priceNum : parsePrice(priceStr),
            img: normalizeImagePath(raw.img),
            quantity: Math.min(MAX_QUANTITY, quantity > 0 ? quantity : 1)
        };
    }

    function mapServerItem(item) {
        return {
            id: String(item.productId || ''),
            title: String(item.title || ''),
            priceStr: item.priceText || '',
            priceNum: item.priceCents / 100,
            img: normalizeImagePath(item.image),
            quantity: item.quantity
        };
    }

    function loadLocal() {
        try {
            const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]');
            items = Array.isArray(saved) ? saved.map(normalizeItem).filter((item) => item.id) : [];
        } catch (error) {
            console.warn('购物车数据读取失败，已重置。', error);
            items = [];
        }
    }

    function saveLocal() {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
        } catch (error) {
            console.warn('购物车数据保存失败。', error);
        }
    }

    function notify() {
        listeners.forEach((listener) => listener());
    }

    function findItem(productId) {
        return items.find((item) => item.id === productId);
    }

    async function loadServerCart() {
        const data = await api.get('/api/cart');
        items = ((data.cart && data.cart.items) || []).map(mapServerItem);
    }

    // 登录态下发现本地仍有未合并的游客购物车时（上次合并失败、或多标签页残留），
    // 自动重试合并；成功后清除本地数据。失败则保留，等待下次 init 再试。
    async function mergePendingLocalCart() {
        let saved = null;
        try {
            saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null');
        } catch (error) {
            saved = null;
        }
        if (!Array.isArray(saved)) {
            return;
        }
        const pending = saved.map(normalizeItem).filter((item) => item.id);
        if (pending.length === 0) {
            clearLocal();
            return;
        }
        try {
            await api.post('/api/cart/merge', {
                items: pending.map((item) => ({ productId: item.id, quantity: item.quantity }))
            });
            await loadServerCart();
            clearLocal();
            console.info('已自动合并上次未同步的游客购物车。');
        } catch (error) {
            console.warn('未同步的游客购物车合并失败，将在下次加载时重试。', error);
        }
    }

    async function init() {
        // 会话请求由 api.session() 记忆化，与 auth-nav 共享同一次 /api/auth/me。
        const user = await api.session();

        mode = user ? 'server' : 'guest';
        if (mode === 'server') {
            try {
                await loadServerCart();
            } catch (error) {
                console.warn('服务端购物车读取失败，已回退为游客模式。', error);
                mode = 'guest';
                loadLocal();
            }
            if (mode === 'server') {
                await mergePendingLocalCart();
            }
        } else {
            loadLocal();
        }
        notify();
        return user;
    }

    async function refresh() {
        if (mode === 'server') {
            await loadServerCart();
        } else {
            loadLocal();
        }
        notify();
    }

    async function addItem(product, quantity) {
        if (!product || !product.id) {
            return;
        }

        const amount = Math.max(MIN_QUANTITY, parseInt(quantity, 10) || 1);
        if (mode === 'server') {
            const data = await api.post('/api/cart/items', { productId: product.id, quantity: amount });
            items = ((data.cart && data.cart.items) || []).map(mapServerItem);
        } else {
            const existing = findItem(product.id);
            if (existing) {
                existing.quantity = Math.min(MAX_QUANTITY, existing.quantity + amount);
            } else {
                items.push(normalizeItem({
                    id: product.id,
                    title: product.title,
                    price: product.price,
                    img: product.img,
                    quantity: amount
                }));
            }
            saveLocal();
        }
        notify();
    }

    async function updateQuantity(productId, quantity) {
        const parsed = parseInt(quantity, 10);
        if (Number.isNaN(parsed)) {
            return;
        }
        // 两种模式统一钳制到 1–99，越界请求不会发到后端。
        const next = Math.min(MAX_QUANTITY, Math.max(MIN_QUANTITY, parsed));

        if (mode === 'server') {
            try {
                const data = await api.patch(`/api/cart/items/${encodeURIComponent(productId)}`, { quantity: next });
                items = ((data.cart && data.cart.items) || []).map(mapServerItem);
            } catch (error) {
                if (error.status === 404) {
                    await refresh();
                    return;
                }
                throw error;
            }
        } else {
            const item = findItem(productId);
            if (item) {
                item.quantity = next;
                saveLocal();
            }
        }
        notify();
    }

    async function removeItem(productId) {
        if (mode === 'server') {
            try {
                const data = await api.del(`/api/cart/items/${encodeURIComponent(productId)}`);
                items = ((data.cart && data.cart.items) || []).map(mapServerItem);
            } catch (error) {
                if (error.status === 404) {
                    await refresh();
                    return;
                }
                throw error;
            }
        } else {
            items = items.filter((item) => item.id !== productId);
            saveLocal();
        }
        notify();
    }

    function getCount() {
        return items.reduce((sum, item) => sum + item.quantity, 0);
    }

    function getSubtotal() {
        return items.reduce((sum, item) => sum + item.priceNum * item.quantity, 0);
    }

    function getTotal() {
        return Math.max(0, getSubtotal());
    }

    // 清除本地游客购物车：登录合并成功后、退出登录时调用。
    function clearLocal() {
        try {
            localStorage.removeItem(STORAGE_KEY);
        } catch (error) {
            /* 忽略 */
        }
        if (mode === 'guest') {
            items = [];
            notify();
        }
    }

    return {
        init,
        refresh,
        addItem,
        updateQuantity,
        removeItem,
        onChange: (listener) => { listeners.add(listener); },
        clearLocal,
        get items() { return items; },
        get mode() { return mode; },
        MIN_QUANTITY,
        MAX_QUANTITY,
        getCount,
        getSubtotal,
        getTotal
    };
})();
