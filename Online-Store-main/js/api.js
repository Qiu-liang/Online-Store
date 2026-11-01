/**
 * 后端 API 封装。
 * 所有请求同源携带会话 Cookie；非 2xx 响应抛出带 status / code 的 Error。
 */
window.api = (function () {
    'use strict';

    async function request(path, options) {
        const config = options || {};
        const init = {
            method: config.method || 'GET',
            credentials: 'same-origin',
            headers: {}
        };

        if (config.body !== undefined) {
            init.headers['Content-Type'] = 'application/json';
            init.body = JSON.stringify(config.body);
        }

        let response;
        try {
            response = await fetch(path, init);
        } catch (error) {
            const networkError = new Error('无法连接服务器，请确认后端服务已启动');
            networkError.code = 'network_error';
            throw networkError;
        }

        let data = null;
        try {
            data = await response.json();
        } catch (error) {
            data = null;
        }

        if (!response.ok) {
            const apiError = new Error(
                (data && data.error && data.error.message) || `请求失败 (${response.status})`
            );
            apiError.status = response.status;
            apiError.code = (data && data.error && data.error.code) || 'request_failed';
            throw apiError;
        }

        return data;
    }

    // HTML 转义：所有插入 innerHTML 的动态文本必须经过它。
    function escapeHtml(text) {
        return String(text == null ? '' : text)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    // CSS url() 值：引号与反斜杠一律百分号编码，防止从 style 属性的字符串中逃逸。
    // 注意 escapeHtml 在 HTML 属性里会被先解码再进入 CSS，不能用于此场景。
    function cssUrl(value) {
        const safe = String(value == null ? '' : value)
            .replace(/\\/g, '%5C')
            .replace(/'/g, '%27')
            .replace(/"/g, '%22')
            .replace(/[\r\n]/g, '');
        return `url('${safe}')`;
    }

    // 当前会话用户：每页只请求一次 /api/auth/me，各模块共享同一个 Promise。
    // 失败（未登录、会话过期或服务不可用）时返回 { user: null, error }，不抛错；
    // 调用方可用 error.status 区分 401 与网络/服务端错误。
    let sessionPromise = null;
    function sessionState() {
        if (!sessionPromise) {
            sessionPromise = request('/api/auth/me')
                .then((data) => ({ user: (data && data.user) || null, error: null }))
                .catch((error) => ({ user: null, error }));
        }
        return sessionPromise;
    }

    function session() {
        return sessionState().then((state) => state.user);
    }

    return {
        request,
        get: (path) => request(path),
        post: (path, body) => request(path, { method: 'POST', body }),
        patch: (path, body) => request(path, { method: 'PATCH', body }),
        del: (path) => request(path, { method: 'DELETE' }),
        escapeHtml,
        cssUrl,
        session,
        sessionState,
        // 金额统一为“分”，展示时转成两位小数。
        formatMoney: (cents) => `¥${(cents / 100).toFixed(2)}`
    };
})();
