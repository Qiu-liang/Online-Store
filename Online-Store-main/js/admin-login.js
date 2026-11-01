/**
 * 后台登录页：进入时检查既有管理员会话，提交后跳转 admin.html。
 * 依赖 window.api（js/api.js）。
 */
(function () {
    'use strict';

    const api = window.api;
    if (!api) {
        console.error('API 模块未正确加载。');
        return;
    }

    const form = document.getElementById('admin-login-form');
    const alertBox = document.getElementById('login-alert');
    const loading = document.getElementById('login-loading');
    const emailInput = document.getElementById('login-email');
    const passwordInput = document.getElementById('login-password');
    const submitButton = document.getElementById('login-submit');

    function showAlert(message) {
        alertBox.textContent = message;
        alertBox.hidden = false;
    }

    function hideAlert() {
        alertBox.hidden = true;
    }

    // 从后台跳转过来时带上 error=not_admin，直接给出无权限提示。
    if (new URLSearchParams(window.location.search).get('error') === 'not_admin') {
        showAlert('当前登录账号没有管理员权限，请使用管理员账号登录。');
    }

    // 已登录管理员直接进入后台；403 提示无权限；401 显示登录表单。
    api.request('/api/admin/auth/me').then(() => {
        window.location.replace('admin.html');
    }).catch((error) => {
        loading.hidden = true;
        if (error.status === 403) {
            showAlert('当前登录账号没有管理员权限，请使用管理员账号登录。');
        }
        form.hidden = false;
        emailInput.focus();
    });

    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        hideAlert();

        const email = emailInput.value.trim();
        const password = passwordInput.value;
        if (!email || !password) {
            showAlert('请输入邮箱和密码。');
            return;
        }

        submitButton.disabled = true;
        try {
            await api.post('/api/admin/auth/login', { email, password });
            window.location.replace('admin.html');
        } catch (error) {
            submitButton.disabled = false;
            showAlert(error.message || '登录失败，请稍后再试。');
            passwordInput.value = '';
            passwordInput.focus();
        }
    });
})();
