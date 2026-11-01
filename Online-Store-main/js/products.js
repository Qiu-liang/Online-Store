/**
 * 产品中心页面逻辑：从后端加载分类与商品，渲染列表、详情弹窗与加入购物车。
 */
(function () {
    'use strict';

    const api = window.api;
    const cart = window.greenCart;
    if (!api || !cart) {
        console.error('API 或购物车模块未正确加载。');
        return;
    }

    const contentContainer = document.getElementById('content-container');
    const modal = document.getElementById('product-detail-modal');
    const closeButton = document.getElementById('close-modal');
    const detailImage = document.getElementById('detail-image');
    const detailTitle = document.getElementById('detail-title');
    const detailRating = document.getElementById('detail-rating');
    const detailDescription = document.getElementById('detail-description');
    const detailPrice = document.getElementById('detail-price');
    const detailMeta = document.getElementById('detail-meta');
    const detailAddButton = document.querySelector('.detail-actions .detail-cta:not(.secondary)');

    const state = {
        categories: [],
        products: [],
        activeCategory: 'vegetable',
        activeProduct: null,
        lastFocusedElement: null
    };

    const escapeHtml = api.escapeHtml;
    const cssUrl = api.cssUrl;

    function categoryTitle(slug) {
        const found = state.categories.find((category) => category.slug === slug);
        return found ? found.title : '';
    }

    function productsOf(category) {
        return state.products.filter((product) => product.category === category);
    }

    function renderStars(rating) {
        let stars = '';

        for (let index = 1; index <= 5; index += 1) {
            if (index <= Math.floor(rating)) {
                stars += '<i class="fas fa-star" aria-hidden="true"></i>';
            } else if (index === Math.ceil(rating) && !Number.isInteger(rating)) {
                stars += '<i class="fas fa-star-half-alt" aria-hidden="true"></i>';
            } else {
                stars += '<i class="far fa-star" aria-hidden="true"></i>';
            }
        }

        return stars;
    }

    function productCard(product) {
        return `
            <article class="product-card fade-in" data-id="${escapeHtml(product.id)}">
                <div class="product-image" style="background-image: ${cssUrl(product.image)}"
                     role="img" aria-label="${escapeHtml(product.title)}"></div>
                <div class="product-info">
                    <h3 class="product-title">${escapeHtml(product.title)}</h3>
                    <div class="rating">
                        ${renderStars(product.rating)}
                        <span>${escapeHtml(product.rating)}</span>
                    </div>
                    <p class="product-description">${escapeHtml(product.description)}</p>
                    <div class="product-price">${escapeHtml(product.priceText)}</div>
                    <button type="button" class="product-cta" data-id="${escapeHtml(product.id)}">
                        <i class="fas fa-shopping-cart" aria-hidden="true"></i> 加入购物车
                    </button>
                </div>
            </article>
        `;
    }

    function renderLoading() {
        contentContainer.innerHTML = `
            <section class="products-container">
                <div class="catalog-loading">
                    <i class="fas fa-spinner fa-spin" aria-hidden="true"></i>
                    <p>正在加载商品…</p>
                </div>
            </section>
        `;
    }

    function renderError(message) {
        contentContainer.innerHTML = `
            <section class="products-container">
                <div class="catalog-error">
                    <i class="fas fa-triangle-exclamation" aria-hidden="true"></i>
                    <h3>商品加载失败</h3>
                    <p>${escapeHtml(message)}</p>
                    <button type="button" class="modern-cta" id="retry-catalog">重新加载</button>
                </div>
            </section>
        `;
        document.getElementById('retry-catalog').addEventListener('click', start);
    }

    function renderProducts(category) {
        const items = productsOf(category);
        const title = categoryTitle(category);

        if (!title || items.length === 0) {
            contentContainer.innerHTML = `
                <section class="products-container">
                    <div class="catalog-error">
                        <i class="fas fa-box-open" aria-hidden="true"></i>
                        <h3>该分类暂无商品</h3>
                        <p>请稍后再试，或选择其他分类。</p>
                    </div>
                </section>
            `;
            return;
        }

        contentContainer.innerHTML = `
            <section class="products-container">
                <h2 class="section-title">
                    <span class="title-main">${escapeHtml(title)}</span>
                </h2>
                <div class="product-grid">
                    ${items.map(productCard).join('')}
                </div>
            </section>
        `;
    }

    function buildMeta(product) {
        if (product.type === '花卉') {
            return `
                <div class="meta-item">
                    <div class="meta-title">产地</div>
                    <div class="meta-value">${escapeHtml(product.origin)}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-title">养护方法</div>
                    <div class="meta-value">${escapeHtml(product.care)}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-title">可选颜色</div>
                    <div class="meta-value">${escapeHtml(product.color)}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-title">花材类型</div>
                    <div class="meta-value">${escapeHtml(product.type)}</div>
                </div>
            `;
        }

        return `
            <div class="meta-item">
                <div class="meta-title">产地</div>
                <div class="meta-value">${escapeHtml(product.origin)}</div>
            </div>
            <div class="meta-item">
                <div class="meta-title">保存方式</div>
                <div class="meta-value">${escapeHtml(product.storage)}</div>
            </div>
            <div class="meta-item">
                <div class="meta-title">主要营养</div>
                <div class="meta-value">${escapeHtml(product.nutrients)}</div>
            </div>
            <div class="meta-item">
                <div class="meta-title">产品类型</div>
                <div class="meta-value">${escapeHtml(product.type)}</div>
            </div>
        `;
    }

    function openProductModal(productId) {
        const product = state.products.find((item) => item.id === productId);
        if (!product) {
            return;
        }

        state.activeProduct = product;
        state.lastFocusedElement = document.activeElement;

        detailImage.style.backgroundImage = cssUrl(product.image);
        detailImage.setAttribute('aria-label', `${product.title}产品图`);
        detailTitle.textContent = product.title;
        detailDescription.textContent = product.detail;
        detailPrice.textContent = product.priceText;
        detailRating.innerHTML = `
            ${renderStars(product.rating)}
            <span>${escapeHtml(product.rating)} (128条评价)</span>
        `;
        detailMeta.innerHTML = buildMeta(product);

        modal.classList.add('active');
        modal.setAttribute('aria-hidden', 'false');
        document.body.style.overflow = 'hidden';
        closeButton.focus();
        // visibility 过渡尚未推进时（低性能设备/后台标签页）同步 focus 可能落空，短暂延迟后补一次。
        if (document.activeElement !== closeButton) {
            setTimeout(() => {
                if (state.activeProduct && document.activeElement !== closeButton) {
                    closeButton.focus();
                }
            }, 80);
        }
    }

    function closeModal() {
        modal.classList.remove('active');
        modal.setAttribute('aria-hidden', 'true');
        document.body.style.overflow = '';
        state.activeProduct = null;

        if (state.lastFocusedElement) {
            state.lastFocusedElement.focus();
            state.lastFocusedElement = null;
        }
    }

    function showToast(message) {
        const toast = document.createElement('div');
        toast.className = 'toast';
        toast.innerHTML = `
            <i class="fas fa-check-circle" aria-hidden="true"></i>
            <span>${escapeHtml(message)}</span>
        `;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.classList.add('is-hiding');
            setTimeout(() => toast.remove(), 400);
        }, 3000);
    }

    async function addToCart(product, button) {
        if (!product) {
            return;
        }

        try {
            await cart.addItem({
                id: product.id,
                title: product.title,
                price: product.priceText,
                img: product.image
            }, 1);

            if (button) {
                button.classList.add('added');
                setTimeout(() => button.classList.remove('added'), 800);
            }
            showToast(`已添加 ${product.title} 到购物车`);
        } catch (error) {
            showToast(error.message || '加入购物车失败');
        }
    }

    contentContainer.addEventListener('click', (event) => {
        const addButton = event.target.closest('.product-cta');
        if (addButton) {
            const product = state.products.find((item) => item.id === addButton.dataset.id);
            addToCart(product, addButton);
            return;
        }

        const card = event.target.closest('.product-card');
        if (card) {
            openProductModal(card.dataset.id);
        }
    });

    document.querySelectorAll('.category-buttons button').forEach((button) => {
        button.addEventListener('click', () => {
            document.querySelectorAll('.category-buttons button').forEach((item) => {
                item.classList.remove('active');
            });
            button.classList.add('active');
            state.activeCategory = button.dataset.category;

            if (state.products.length > 0) {
                renderProducts(state.activeCategory);
            }
        });
    });

    detailAddButton.addEventListener('click', () => {
        addToCart(state.activeProduct, detailAddButton);
    });

    closeButton.addEventListener('click', closeModal);

    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && modal.classList.contains('active')) {
            closeModal();
        }
    });

    async function start() {
        renderLoading();

        try {
            const [categoryData, productData] = await Promise.all([
                api.get('/api/categories'),
                api.get('/api/products')
            ]);
            state.categories = (categoryData && categoryData.categories) || [];
            state.products = (productData && productData.products) || [];
            renderProducts(state.activeCategory);
        } catch (error) {
            renderError(error.message || '网络异常，请稍后再试');
        }
    }

    // 购物车模块就绪后再渲染，避免登录状态下首次加购落入本地存储。
    cart.init().finally(() => {
        start();
    });
})();
