/**
 * 首页逻辑：按钮导航与主视觉视差滚动。
 */
(function () {
    'use strict';

    // 带 data-goto 的按钮统一在这里跳转（替代内联 onclick，配合 CSP 限制内联脚本）。
    document.addEventListener('click', (event) => {
        const target = event.target.closest('[data-goto]');
        if (target) {
            window.location.href = target.dataset.goto;
        }
    });

    const hero = document.querySelector('.hero');
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    if (!hero || prefersReducedMotion) {
        return;
    }

    window.addEventListener('scroll', () => {
        hero.style.backgroundPositionY = `${window.scrollY * 0.5}px`;
    }, { passive: true });
})();
