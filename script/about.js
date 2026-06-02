// 数字递增动画
const counters = document.querySelectorAll('.stat-number');
counters.forEach(counter => {
    const target = +counter.dataset.count;
    const increment = target / 100;
    let current = 0;
    
    const updateCount = () => {
        if(current < target) {
            current += increment;
            counter.textContent = Math.ceil(current);
            setTimeout(updateCount, 20);
        } else {
            counter.textContent = target;
        }
    }
    updateCount();
});

// 添加视差滚动效果
window.addEventListener('scroll', () => {
    const scrolled = window.pageYOffset;
    document.querySelector('.about-hero').style.backgroundPositionY = scrolled * 0.5 + 'px';
});

// 团队卡片悬停效果
document.querySelectorAll('.member-card').forEach(card => {
    card.addEventListener('mousemove', (e) => {
        const rect = card.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        card.style.transform = `perspective(1000px) rotateX(${(y - rect.height/2)/20}deg) rotateY(${-(x - rect.width/2)/20}deg)`;
    });
    
    card.addEventListener('mouseleave', () => {
        card.style.transform = 'none';
    });
});