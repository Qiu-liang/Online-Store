/**
 * 登录 / 注册页面逻辑：调用后端接口、合并游客购物车并跳转。
 */
(function () {
    'use strict';

    const api = window.api;
    if (!api) {
        console.error('API 模块未正确加载。');
        return;
    }

    const tabs = document.querySelectorAll('.auth-tab');
    const forms = document.querySelectorAll('.auth-form');
    const message = document.getElementById('form-message');

    function switchForm(formType) {
        hideMessage();
        tabs.forEach((tab) => {
            const isActive = tab.dataset.form === formType;
            tab.classList.toggle('active', isActive);
            tab.setAttribute('aria-pressed', String(isActive));
        });

        forms.forEach((form) => {
            form.classList.toggle('active', form.id === `${formType}Form`);
        });
    }

    tabs.forEach((tab) => {
        tab.addEventListener('click', () => switchForm(tab.dataset.form));
    });

    function showMessage(text, type) {
        message.textContent = text;
        message.className = `form-message ${type}`;
        message.hidden = false;
    }

    function hideMessage() {
        message.hidden = true;
    }

    function setPending(form, pending) {
        const button = form.querySelector('button[type="submit"]');
        button.disabled = pending;
    }

    // 只接受站内 .html 路径，避免开放跳转。
    function nextTarget() {
        const next = new URLSearchParams(window.location.search).get('next') || '';
        return /^[a-z]+\.html$/.test(next) ? next : 'account.html';
    }

    // 通过购物车数据层清除本地游客购物车；cart-store 未加载时兜底直删。
    function clearGuestCart() {
        if (window.greenCart) {
            window.greenCart.clearLocal();
            return;
        }
        try {
            localStorage.removeItem('greenCart');
        } catch (error) {
            /* 忽略 */
        }
    }

    // 登录/注册成功后，把游客购物车合并到服务端。
    async function mergeGuestCart() {
        let parsed = null;
        try {
            parsed = JSON.parse(localStorage.getItem('greenCart') || 'null');
        } catch (error) {
            clearGuestCart();
            return;
        }

        const items = (Array.isArray(parsed) ? parsed : [])
            .filter((item) => item && item.id)
            .map((item) => ({
                productId: String(item.id),
                quantity: Math.max(1, parseInt(item.quantity, 10) || 1)
            }));

        if (!items.length) {
            clearGuestCart();
            return;
        }

        try {
            await api.post('/api/cart/merge', { items });
            clearGuestCart();
        } catch (error) {
            // 合并失败不阻塞登录，本地购物车保留；
            // cart-store 会在下次页面 init 时检测到未同步数据并自动重试合并。
            console.warn('游客购物车合并失败：', error);
        }
    }

    forms.forEach((form) => {
        form.addEventListener('submit', async (event) => {
            event.preventDefault();
            hideMessage();

            const isRegister = form.id === 'registerForm';
            if (isRegister) {
                const password = form.querySelector('input[name="password"]');
                const confirmPassword = form.querySelector('input[name="confirmPassword"]');
                if (password.value !== confirmPassword.value) {
                    showMessage('两次密码输入不一致', 'error');
                    confirmPassword.focus();
                    return;
                }
            }

            setPending(form, true);
            try {
                if (isRegister) {
                    await api.post('/api/auth/register', {
                        username: form.username.value.trim(),
                        email: form.email.value.trim(),
                        password: form.password.value
                    });
                } else {
                    await api.post('/api/auth/login', {
                        email: form.email.value.trim(),
                        password: form.password.value
                    });
                }

                await mergeGuestCart();
                window.location.href = nextTarget();
            } catch (error) {
                showMessage(error.message || '操作失败，请稍后再试', 'error');
                setPending(form, false);
            }
        });
    });
})();
