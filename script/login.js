// 表单切换逻辑
const tabs = document.querySelectorAll('.auth-tab');
const forms = document.querySelectorAll('.auth-form');

tabs.forEach(tab => {
    tab.addEventListener('click', () => {
        const formType = tab.dataset.form;
        
        // 更新选项卡状态
        tabs.forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        
        // 切换表单显示
        forms.forEach(form => {
            form.classList.remove('active');
            if(form.id === `${formType}Form`) {
                setTimeout(() => form.classList.add('active'), 10);
            }
        });
    });
});

// 表单验证
document.querySelectorAll('form').forEach(form => {
    form.addEventListener('submit', (e) => {
        e.preventDefault();
        
        if(form.id === 'registerForm') {
            const password = form.querySelector('input[type="password"]');
            const confirm = form.querySelectorAll('input[type="password"]')[1];
            if(password.value !== confirm.value) {
                alert('两次密码输入不一致');
                return;
            }
        }
        
        // 这里可以添加AJAX提交逻辑
        console.log('表单提交成功');
    });
});