/**
 * 购物车页面逻辑：渲染商品、数量增减、删除、金额汇总与结算。
 */
(function () {
    'use strict';

    const api = window.api;
    const cart = window.greenCart;
    if (!api || !cart) {
        console.error('API 或购物车模块未正确加载。');
        return;
    }

    const itemsContainer = document.getElementById('cart-items-container');
    const cartContainer = document.querySelector('.cart-container');
    const countLabel = document.getElementById('cart-count');
    const subtotalLabel = document.getElementById('subtotal');
    const totalLabel = document.getElementById('total');
    const summaryCard = document.getElementById('summary-card');
    const checkoutButton = document.getElementById('checkout-btn');
    const checkoutPanel = document.getElementById('checkout-panel');
    const checkoutForm = document.getElementById('checkout-form');
    const checkoutMessage = document.getElementById('checkout-message');
    const checkoutCancel = document.getElementById('checkout-cancel');
    const checkoutSubmit = checkoutForm.querySelector('button[type="submit"]');

    // 与 api.formatMoney（分转元）区分：这里接收的是元为单位的浮点数。
    function formatYuan(value) {
        return `¥${value.toFixed(2)}`;
    }

    function cartItemMarkup(item) {
        const safeId = api.escapeHtml(item.id);
        const safeTitle = api.escapeHtml(item.title);
        const safePrice = api.escapeHtml(item.priceStr);
        const minusDisabled = item.quantity <= cart.MIN_QUANTITY ? 'disabled' : '';
        const plusDisabled = item.quantity >= cart.MAX_QUANTITY ? 'disabled' : '';

        return `
            <div class="cart-item">
                <img class="item-image" src="${api.escapeHtml(item.img)}" alt="${safeTitle}"
                     loading="lazy" decoding="async">
                <div class="item-details">
                    <h3 class="product-title">${safeTitle}</h3>
                    <div class="product-price">${safePrice}</div>
                </div>
                <div class="quantity-control">
                    <button type="button" class="quantity-btn minus" data-id="${safeId}" ${minusDisabled}
                            aria-label="减少 ${safeTitle} 数量">-</button>
                    <span>${item.quantity}</span>
                    <button type="button" class="quantity-btn plus" data-id="${safeId}" ${plusDisabled}
                            aria-label="增加 ${safeTitle} 数量">+</button>
                </div>
                <button type="button" class="product-cta delete" data-id="${safeId}">删除</button>
            </div>
        `;
    }

    function emptyCartMarkup() {
        const error = cart.sessionError;
        if (error && error.status !== 401) {
            return `
                <div class="empty-cart">
                    <i class="fas fa-triangle-exclamation" aria-hidden="true"></i>
                    <h3>购物车加载失败</h3>
                    <p>${api.escapeHtml(error.message || '无法连接服务器，请稍后再试')}</p>
                    <button type="button" class="shop-btn" id="reload-cart">重新加载</button>
                </div>
            `;
        }

        if (cart.mode === 'guest') {
            return `
                <div class="empty-cart">
                    <i class="fas fa-shopping-cart" aria-hidden="true"></i>
                    <h3>登录后查看您的购物车</h3>
                    <p>当前未登录。若您之前登录过，购物车中的商品仍保存在账户中。</p>
                    <div class="empty-cart-actions">
                        <a href="login.html?next=shopping.html" class="shop-btn">去登录</a>
                        <a href="products.html" class="shop-btn shop-btn-secondary">去逛逛</a>
                    </div>
                </div>
            `;
        }

        return `
            <div class="empty-cart">
                <i class="fas fa-shopping-cart" aria-hidden="true"></i>
                <h3>您的购物车是空的</h3>
                <p>快去挑选新鲜有机的蔬菜水果，丰富您的健康生活吧！</p>
                <a href="products.html" class="shop-btn">去逛逛</a>
            </div>
        `;
    }

    function renderCart() {
        countLabel.textContent = `您已选择 ${cart.getCount()} 件商品`;

        if (cart.items.length === 0) {
            cartContainer.classList.add('cart-empty');
            cartContainer.classList.remove('checkout-open');
            itemsContainer.innerHTML = emptyCartMarkup();
            summaryCard.hidden = true;
            checkoutPanel.hidden = true;
            return;
        }

        cartContainer.classList.remove('cart-empty');
        const warning = cart.sessionError && cart.sessionError.status !== 401
            ? `<p class="cart-warning">${api.escapeHtml(cart.sessionError.message || '无法连接服务器，显示的是本地购物车，可能不是最新数据。')}</p>`
            : '';
        itemsContainer.innerHTML = warning + cart.items.map(cartItemMarkup).join('');
        summaryCard.hidden = false;
        subtotalLabel.textContent = formatYuan(cart.getSubtotal());
        totalLabel.textContent = formatYuan(cart.getTotal());
    }

    itemsContainer.addEventListener('click', (event) => {
        if (event.target.closest('#reload-cart')) {
            window.location.reload();
            return;
        }

        const button = event.target.closest('button[data-id]');
        if (!button) {
            return;
        }

        const productId = button.dataset.id;
        const item = cart.items.find((entry) => entry.id === productId);
        if (!item) {
            return;
        }

        if (button.classList.contains('minus')) {
            cart.updateQuantity(productId, item.quantity - 1).catch(showError);
        } else if (button.classList.contains('plus')) {
            cart.updateQuantity(productId, item.quantity + 1).catch(showError);
        } else if (button.classList.contains('delete')) {
            cart.removeItem(productId).catch(showError);
        }
    });

    function showError(error) {
        alert(error.message || '操作失败，请稍后再试');
    }

    checkoutButton.addEventListener('click', () => {
        if (cart.getCount() === 0) {
            alert('您的购物车是空的，请先添加商品');
            return;
        }

        if (cart.mode === 'guest') {
            window.location.href = 'login.html?next=shopping.html';
            return;
        }

        cartContainer.classList.add('checkout-open');
        checkoutPanel.hidden = false;
        checkoutPanel.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });

    checkoutCancel.addEventListener('click', () => {
        cartContainer.classList.remove('checkout-open');
        checkoutPanel.hidden = true;
        checkoutMessage.hidden = true;
    });

    checkoutForm.addEventListener('submit', async (event) => {
        event.preventDefault();
        checkoutMessage.hidden = true;
        checkoutSubmit.disabled = true;

        try {
            const data = await api.post('/api/orders', {
                shippingName: checkoutForm.shippingName.value.trim(),
                shippingPhone: checkoutForm.shippingPhone.value.trim(),
                shippingAddress: checkoutForm.shippingAddress.value.trim()
            });
            window.location.href = `account.html?order=${encodeURIComponent(data.order.id)}`;
        } catch (error) {
            if (error.status === 401) {
                // 会话过期：先去登录，登录后自动回到购物车页。
                window.location.href = 'login.html?next=shopping.html';
                return;
            }
            checkoutMessage.textContent = error.message || '下单失败，请稍后再试';
            checkoutMessage.className = 'form-message error';
            checkoutMessage.hidden = false;
            checkoutSubmit.disabled = false;
        }
    });

    window.addEventListener('storage', (event) => {
        if (cart.mode === 'guest' && event.key === 'greenCart') {
            cart.refresh().catch(() => { /* 忽略 */ });
        }
    });

    (async () => {
        await cart.init();
        renderCart();
    })();
    cart.onChange(renderCart);
})();
