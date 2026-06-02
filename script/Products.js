// 产品数据
        const products = {
            vegetable: [
                { 
                    id: 'veg-1',
                    img: '../images/Products/tomato.jpg', 
                    title: '番茄', 
                    desc: '每颗番茄都经过严格筛选', 
                    price: '¥26.8/500g',
                    detail: '我们的有机番茄种植在阳光充足的山谷中，采用纯天然有机肥料，无农药残留。完全自然成熟，口感酸甜适中，富含维生素C和番茄红素。采摘后24小时内送达，保证新鲜度。',
                    origin: '山东寿光',
                    storage: '冷藏保存7天',
                    nutrients: '维生素C、番茄红素、钾',
                    rating: 4.8,
                    type: '蔬菜'
                },
                { 
                    id: 'veg-2',
                    img: '../images/Products/cucumber.jpg', 
                    title: '黄瓜', 
                    desc: '新鲜采摘，脆嫩多汁', 
                    price: '¥18.5/500g',
                    detail: '清晨采摘的有机黄瓜，脆嫩多汁，口感清爽。采用生态种植方式，无农药、无化肥，富含维生素K和多种矿物质。适合凉拌、榨汁或直接食用。',
                    origin: '河北保定',
                    storage: '冷藏保存5天',
                    nutrients: '维生素K、钾、膳食纤维',
                    rating: 4.6,
                    type: '蔬菜'
                },
                { 
                    id: 'veg-3',
                    img: '../images/Products/carrots.jpg', 
                    title: '胡萝卜', 
                    desc: '自然甜脆，富含胡萝卜素', 
                    price: '¥15.9/500g',
                    detail: '我们的有机胡萝卜生长在富含矿物质的土壤中，自然甜脆，富含β-胡萝卜素。采用轮作种植方式，保证土壤健康。生食清脆甘甜，烹饪后更显风味。',
                    origin: '内蒙古赤峰',
                    storage: '冷藏保存10天',
                    nutrients: 'β-胡萝卜素、维生素A、钾',
                    rating: 4.7,
                    type: '蔬菜'
                },
                { 
                    id: 'veg-4',
                    img: '../images/Products/broccoli.jpg', 
                    title: '西兰花', 
                    desc: '营养丰富，口感清脆', 
                    price: '¥22.5/500g',
                    detail: '有机西兰花采用无公害种植方式，富含维生素C、K和叶酸。花球紧密，颜色鲜绿，口感清脆。建议蒸煮保留最大营养价值。',
                    origin: '云南昆明',
                    storage: '冷藏保存5天',
                    nutrients: '维生素C、维生素K、叶酸',
                    rating: 4.9,
                    type: '蔬菜'
                }
            ],
            fruit: [
                { 
                    id: 'fruit-1',
                    img: '../images/Products/bluberry.jpg',
                    title: '蓝莓',
                    desc: '来自海拔1500米的生态种植园',
                    price: '¥58.0/盒',
                    detail: '我们的蓝莓种植在海拔1500米的高原生态种植园，昼夜温差大，果实甜度高。采用物理防虫方式，无农药残留。富含花青素和抗氧化物质，有助于保护视力、延缓衰老。',
                    origin: '云南曲靖',
                    storage: '冷藏保存7天',
                    nutrients: '花青素、维生素C、锰',
                    rating: 4.9,
                    type: '水果'
                },
                { 
                    id: 'fruit-2',
                    img: '../images/Products/strawberry.jpg',
                    title: '草莓',
                    desc: '新鲜采摘，香甜多汁',
                    price: '¥45.0/500g',
                    detail: '清晨采摘的有机草莓，色泽鲜红，香气浓郁。采用温室有机种植，严格控制温度和湿度，保证草莓自然成熟。口感香甜多汁，富含维生素C和抗氧化物质。',
                    origin: '辽宁丹东',
                    storage: '冷藏保存3天',
                    nutrients: '维生素C、叶酸、锰',
                    rating: 4.7,
                    type: '水果'
                },
                { 
                    id: 'fruit-3',
                    img: '../images/Products/grape.jpg',
                    title: '葡萄',
                    desc: '无籽多汁，果香浓郁',
                    price: '¥38.5/500g',
                    detail: '精选无籽葡萄品种，果粒饱满，皮薄肉厚，甜度高。采用有机种植方式，严格控制采摘时间，保证最佳成熟度。富含花青素和多种维生素。',
                    origin: '新疆吐鲁番',
                    storage: '冷藏保存5天',
                    nutrients: '花青素、维生素K、钾',
                    rating: 4.8,
                    type: '水果'
                }
            ],
            flowers: [
                { 
                    id: 'flower-1',
                    img: '../images/Products/rose.jpg',
                    title: '玫瑰',
                    desc: '自然生长，每束玫瑰都经过严格筛选',
                    price: '¥200/扎',
                    detail: '我们的玫瑰来自云南优质花卉基地，采用无土栽培技术，花朵饱满，颜色鲜艳，花期持久。每支玫瑰长度60-70cm，花朵直径8-10cm，经过严格筛选，保证品质。',
                    origin: '云南昆明',
                    care: '每天换水，斜剪花茎',
                    color: '红色、粉色、香槟色',
                    rating: 4.8,
                    type: '花卉'
                },
                { 
                    id: 'flower-2',
                    img: '../images/Products/lily.jpg',
                    title: '百合',
                    desc: '花香浓郁，象征纯洁高雅',
                    price: '¥180/扎',
                    detail: '精选多头百合，每支3-5个花头，花朵大而饱满，花香浓郁持久。百合象征纯洁高雅，是家居装饰和送礼的绝佳选择。',
                    origin: '福建漳州',
                    care: '去除花蕊，避免阳光直射',
                    color: '白色、粉色、黄色',
                    rating: 4.7,
                    type: '花卉'
                },
                { 
                    id: 'flower-3',
                    img: '../images/Products/carnation.jpg',
                    title: '康乃馨',
                    desc: '花色丰富，花期长久',
                    price: '¥150/扎',
                    detail: '康乃馨花朵丰满，色彩丰富，花期长达2周。象征母爱和敬意，是节日送礼的经典选择。采用先进保鲜技术，确保花朵新鲜送达。',
                    origin: '浙江杭州',
                    care: '避免高温环境，定期换水',
                    color: '红色、粉色、白色、紫色',
                    rating: 4.6,
                    type: '花卉'
                }
            ]
        };

        // 标题配置数据
        const categoryTitles = {
            vegetable: {
                main: '田园鲜蔬·自然之选',
            },
            fruit: {
                main: '四季鲜果·阳光之味',
            },
            flowers: {
                main: '花漾时光·生活美学',
            }
        };

        // 获取DOM元素
        const contentContainer = document.getElementById('content-container');
        const modal = document.getElementById('product-detail-modal');
        const closeBtn = document.getElementById('close-modal');
        
        // 渲染产品列表
        function renderProducts(category) {
            const container = document.createElement('div');
            container.className = 'products-container';
            container.innerHTML = `
                <h2 class="section-title">
                    <span class="title-main">${categoryTitles[category].main}</span>
                </h2>
                <div class="product-grid" id="product-grid">
                    ${products[category].map(item => `
                        <article class="product-card fade-in" data-id="${item.id}">
                            <div class="product-image" style="background-image: url('${item.img}')"></div>
                            <div class="product-info">
                                <h3>${item.title}</h3>
                                <div class="rating">
                                    ${renderStars(item.rating)}
                                    <span>${item.rating}</span>
                                </div>
                                <p>${item.desc}</p>
                                <div class="product-price">${item.price}</div>
                                <button class="product-cta"><i class="fas fa-shopping-cart"></i> 加入购物车</button>
                            </div>
                        </article>
                    `).join('')}
                </div>
            `;
            
            // 清空容器并添加新内容
            contentContainer.innerHTML = '';
            contentContainer.appendChild(container);
            
            // 为产品卡片添加事件监听
            document.querySelectorAll('.product-card').forEach(card => {
                card.addEventListener('click', function(e) {
                    if (!e.target.classList.contains('product-cta')) {
                        const productId = this.getAttribute('data-id');
                        openProductModal(productId);
                    }
                });
            });
        }
        
        // 渲染星级评分
        function renderStars(rating) {
            let stars = '';
            for (let i = 1; i <= 5; i++) {
                if (i <= Math.floor(rating)) {
                    stars += '<i class="fas fa-star"></i>';
                } else if (i === Math.ceil(rating) && !Number.isInteger(rating)) {
                    stars += '<i class="fas fa-star-half-alt"></i>';
                } else {
                    stars += '<i class="far fa-star"></i>';
                }
            }
            return stars;
        }
        
        // 打开产品模态框
        function openProductModal(productId) {
            // 在所有产品中查找匹配的产品
            let product = null;
            for (const category in products) {
                const found = products[category].find(p => p.id === productId);
                if (found) {
                    product = found;
                    break;
                }
            }
            
            if (!product) return;
            
            // 填充模态框内容
            document.getElementById('detail-image').style.backgroundImage = `url('${product.img}')`;
            document.getElementById('detail-title').textContent = product.title;
            document.getElementById('detail-description').textContent = product.detail;
            document.getElementById('detail-price').textContent = product.price;
            
            // 渲染评分
            document.getElementById('detail-rating').innerHTML = `
                ${renderStars(product.rating)}
                <span>${product.rating} (128条评价)</span>
            `;
            
            // 渲染元数据
            const isFlower = product.type === '花卉';
            document.getElementById('detail-meta').innerHTML = `
                <div class="meta-item">
                    <div class="meta-title">产地</div>
                    <div class="meta-value">${product.origin}</div>
                </div>
                ${
                    isFlower 
                    ? `
                        <div class="meta-item">
                            <div class="meta-title">养护方法</div>
                            <div class="meta-value">${product.care}</div>
                        </div>
                        <div class="meta-item">
                            <div class="meta-title">可选颜色</div>
                            <div class="meta-value">${product.color}</div>
                        </div>
                        <div class="meta-item">
                            <div class="meta-title">花材类型</div>
                            <div class="meta-value">${product.type}</div>
                        </div>
                    `
                    : `
                        <div class="meta-item">
                            <div class="meta-title">保存方式</div>
                            <div class="meta-value">${product.storage}</div>
                        </div>
                        <div class="meta-item">
                            <div class="meta-title">主要营养</div>
                            <div class="meta-value">${product.nutrients}</div>
                        </div>
                        <div class="meta-item">
                            <div class="meta-title">产品类型</div>
                            <div class="meta-value">${product.type}</div>
                        </div>
                    `
                }
            `;
            
            // 打开模态框
            modal.classList.add('active');
            document.body.style.overflow = 'hidden';
        }
        
        // 关闭模态框
        function closeModal() {
            modal.classList.remove('active');
            document.body.style.overflow = 'auto';
        }
        
        // 事件监听器
        closeBtn.addEventListener('click', closeModal);
        
        modal.addEventListener('click', function(e) {
            if (e.target === modal) {
                closeModal();
            }
        });
        
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && modal.classList.contains('active')) {
                closeModal();
            }
        });
        
        // 分类按钮事件
        document.querySelectorAll('.category-buttons button').forEach(btn => {
            btn.addEventListener('click', function() {
                // 更新按钮状态
                document.querySelectorAll('.category-buttons button').forEach(b => 
                    b.classList.remove('active'));
                this.classList.add('active');
                
                // 渲染对应分类的产品
                renderProducts(this.dataset.category);
            });
        });
        
        // 加入购物车功能
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('product-cta') || 
                (e.target.closest('.detail-cta') && !e.target.closest('.secondary'))) {
                let productTitle;
                if (e.target.closest('.product-card')) {
                    productTitle = e.target.closest('.product-card').querySelector('h3').textContent;
                } else if (e.target.closest('.product-detail-modal')) {
                    productTitle = document.getElementById('detail-title').textContent;
                }
                
                // 添加购物车动画
                const button = e.target.closest('button');
                button.classList.add('added');
                setTimeout(() => {
                    button.classList.remove('added');
                }, 800);
                
                // 显示添加成功消息
                showNotification(`已添加 ${productTitle} 到购物车`);
            }
        });
        
        // 显示通知
        function showNotification(message) {
            const notification = document.createElement('div');
            notification.style.cssText = `
                position: fixed;
                bottom: 30px;
                right: 30px;
                background: #2c7744;
                color: white;
                padding: 15px 25px;
                border-radius: 8px;
                box-shadow: 0 5px 15px rgba(0,0,0,0.2);
                z-index: 1000;
                display: flex;
                align-items: center;
                gap: 10px;
                animation: fadeIn 0.3s ease forwards;
            `;
            notification.innerHTML = `
                <i class="fas fa-check-circle"></i> ${message}
            `;
            document.body.appendChild(notification);
            
            setTimeout(() => {
                notification.style.animation = 'fadeOut 0.3s ease forwards';
                setTimeout(() => {
                    document.body.removeChild(notification);
                }, 300);
            }, 3000);
        }
        
        // 添加fadeIn和fadeOut动画
        const style = document.createElement('style');
        style.innerHTML = `
            @keyframes fadeIn {
                from { opacity: 0; transform: translateY(20px); }
                to { opacity: 1; transform: translateY(0); }
            }
            @keyframes fadeOut {
                from { opacity: 1; transform: translateY(0); }
                to { opacity: 0; transform: translateY(20px); }
            }
        `;
        document.head.appendChild(style);
        
        // 初始化页面
        renderProducts('vegetable');