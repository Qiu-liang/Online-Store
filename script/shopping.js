// 购物车数据结构 
        const cart = {
            items: [],
            discount: 0,
            shipping: 0,
            
            // 添加商品到购物车
            addItem: function(product, quantity = 1) {
                const existingItem = this.items.find(item  => item.id  === product.id); 
                
                if (existingItem) {
                    existingItem.quantity  += quantity;
                } else {
                    // 从价格字符串中提取数值 
                    const priceMatch = product.price.match(/[\d.]+/); 
                    const priceNum = priceMatch ? parseFloat(priceMatch[0]) : 0;
                    
                    this.items.push({ 
                        id: product.id, 
                        title: product.title, 
                        priceStr: product.price, 
                        priceNum: priceNum,
                        img: product.img, 
                        quantity: quantity
                    });
                }
                this.saveToLocalStorage(); 
                this.updateUI(); 
            },
            
            // 更新商品数量 
            updateQuantity: function(productId, newQuantity) {
                if (newQuantity < 1) return;
                
                const item = this.items.find(item  => item.id  === productId);
                if (item) {
                    item.quantity  = parseInt(newQuantity);
                    this.saveToLocalStorage(); 
                    this.updateUI(); 
                }
            },
            
            // 删除商品 
            removeItem: function(productId) {
                this.items  = this.items.filter(item  => item.id  !== productId);
                this.saveToLocalStorage(); 
                this.updateUI(); 
            },
            
            // 计算小计
            calculateSubtotal: function() {
                return this.items.reduce((sum,  item) => {
                    return sum + (item.priceNum  * item.quantity); 
                }, 0);
            },
            
            // 计算总计 
            calculateTotal: function() {
                const subtotal = this.calculateSubtotal(); 
                return subtotal - this.discount  + this.shipping; 
            },
            
            // 保存到本地存储
            saveToLocalStorage: function() {
                localStorage.setItem('greenCart',  JSON.stringify(this.items)); 
            },
            
            // 从本地存储加载
            loadFromLocalStorage: function() {
                const savedCart = localStorage.getItem('greenCart'); 
                if (savedCart) {
                    this.items  = JSON.parse(savedCart); 
                }
            },
            
            // 更新购物车UI
            updateUI: function() {
                const cartItemsContainer = document.getElementById('cart-items-container'); 
                const cartCount = document.getElementById('cart-count'); 
                const subtotalEl = document.getElementById('subtotal'); 
                const totalEl = document.getElementById('total'); 
                const summaryCard = document.getElementById('summary-card'); 
                const checkoutBtn = document.getElementById('checkout-btn'); 
                
                // 更新商品数量显示 
                const totalItems = this.items.reduce((sum,  item) => sum + item.quantity,  0);
                cartCount.textContent  = `您已选择${totalItems}件商品`;
                
                // 清空购物车容器
                cartItemsContainer.innerHTML  = '';
                
                if (this.items.length  === 0) {
                    // 购物车为空时显示
                    cartItemsContainer.innerHTML  = `
                        <div class="empty-cart">
                            <i class="fas fa-shopping-cart"></i>
                            <h3>您的购物车是空的</h3>
                            <p>快去挑选新鲜有机的蔬菜水果，丰富您的健康生活吧！</p>
                            <a href="./products.html"  class="shop-btn">去逛逛</a>
                        </div>
                    `;
                    summaryCard.style.display  = 'none';
                    return;
                } else {
                    summaryCard.style.display  = 'block';
                }
                
                // 渲染购物车中的商品
                this.items.forEach(item  => {
                    const cartItemEl = document.createElement('div'); 
                    cartItemEl.className  = 'cart-item';
                    cartItemEl.innerHTML  = `
                        <div class="item-image" style="background-image: url('${item.img}')"></div> 
                        <div class="item-details">
                            <h3 class="product-title">${item.title}</h3> 
                            <div class="product-price">${item.priceStr}</div> 
                        </div>
                        <div class="quantity-control">
                            <button class="quantity-btn minus" data-id="${item.id}">-</button> 
                            <span>${item.quantity}</span> 
                            <button class="quantity-btn plus" data-id="${item.id}">+</button> 
                        </div>
                        <button class="product-cta delete" data-id="${item.id}"> 删除</button>
                    `;
                    cartItemsContainer.appendChild(cartItemEl); 
                });
                
                // 更新价格摘要
                const subtotal = this.calculateSubtotal(); 
                const total = this.calculateTotal(); 
                
                subtotalEl.textContent  = `¥${subtotal.toFixed(2)}`; 
                totalEl.textContent  = `¥${total.toFixed(2)}`; 
            },
            
            // 初始化购物车事件监听 
            initEventListeners: function() {
                // 委托事件处理 
                document.addEventListener('click',  (event) => {
                    const target = event.target; 
                    
                    // 减少数量 
                    if (target.classList.contains('minus'))  {
                        const productId = target.dataset.id; 
                        const item = this.items.find(item  => item.id  === productId);
                        if (item && item.quantity  > 1) {
                            this.updateQuantity(productId,  item.quantity  - 1);
                        }
                    }
                    
                    // 增加数量 
                    if (target.classList.contains('plus'))  {
                        const productId = target.dataset.id; 
                        const item = this.items.find(item  => item.id  === productId);
                        if (item) {
                            this.updateQuantity(productId,  item.quantity  + 1);
                        }
                    }
                    
                    // 删除商品 
                    if (target.classList.contains('delete'))  {
                        const productId = target.dataset.id; 
                        this.removeItem(productId); 
                    }
                });
                
                // 结算按钮 
                document.getElementById('checkout-btn').addEventListener('click',  () => {
                    if (this.items.length  === 0) {
                        alert('您的购物车是空的，请先添加商品');
                        return;
                    }
                    
                    alert('结算功能开发中！总金额：¥' + this.calculateTotal().toFixed(2)); 
                });
            }
        };
 
        // 页面加载时初始化购物车
        document.addEventListener('DOMContentLoaded',  () => {
            cart.loadFromLocalStorage(); 
            cart.updateUI(); 
            cart.initEventListeners(); 
        });