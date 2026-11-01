/**
 * 关于我们页面逻辑：数字递增、视差滚动与团队卡片倾斜效果。
 */
(function () {
    'use strict';

    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    function animateCounter(counter) {
        const target = Number(counter.dataset.count) || 0;
        const suffix = counter.textContent.replace(/[\d,]/g, '');

        if (prefersReducedMotion) {
            counter.textContent = `${target}${suffix}`;
            return;
        }

        const duration = 1500;
        const startTime = performance.now();

        function tick(now) {
            const progress = Math.min((now - startTime) / duration, 1);
            const value = Math.floor(progress * target);
            counter.textContent = `${value}${suffix}`;

            if (progress < 1) {
                requestAnimationFrame(tick);
            } else {
                counter.textContent = `${target}${suffix}`;
            }
        }

        requestAnimationFrame(tick);
    }

    document.querySelectorAll('.stat-number').forEach(animateCounter);

    if (!prefersReducedMotion) {
        const hero = document.querySelector('.about-hero');
        if (hero) {
            window.addEventListener('scroll', () => {
                hero.style.backgroundPositionY = `${window.scrollY * 0.5}px`;
            }, { passive: true });
        }

        document.querySelectorAll('.member-card').forEach((card) => {
            card.addEventListener('mousemove', (event) => {
                const rect = card.getBoundingClientRect();
                const x = event.clientX - rect.left;
                const y = event.clientY - rect.top;

                card.style.transform = `perspective(1000px) rotateX(${(y - rect.height / 2) / 20}deg) rotateY(${-(x - rect.width / 2) / 20}deg)`;
            });

            card.addEventListener('mouseleave', () => {
                card.style.transform = 'none';
            });
        });
    }
})();
