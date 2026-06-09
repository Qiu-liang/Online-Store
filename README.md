# 绿源农业 Online Store

一个以“绿源农业”为主题的中文静态电商网站项目，展示了农产品、花卉、关于我们、购物车和登录注册等核心页面。项目主要使用原生 HTML、CSS 和 JavaScript 实现，界面风格偏向清新自然，适合用于课程作业、前端练习或静态站点展示。

## 项目特点

- 首页展示品牌主视觉、卖点卡片和当季精选内容
- 产品中心按分类展示新鲜蔬菜、时令水果和园艺花卉
- 产品详情支持弹窗查看，包含评分、产地、保存方式等信息
- 购物车支持本地缓存、数量增减、删除商品和金额汇总
- 关于我们页面展示品牌发展历程、数据统计和核心团队
- 登录注册页面支持表单切换与基础校验
- 页面整体使用 Font Awesome 图标增强视觉表现

## 页面说明

- 首页：`html/index.html`
- 产品中心：`html/products.html`
- 关于我们：`html/about.html`
- 购物车：`html/shopping.html`
- 登录注册：`html/login.html`

## 技术栈

- HTML5
- CSS3
- 原生 JavaScript
- Font Awesome 图标库

## 目录结构

```text
css/
  about.css
  all.min.css
  footer.css
  index.css
  login.css
  nav.css
  Products.css
  shopping.css
html/
  about.html
  index.html
  login.html
  products.html
  shopping.html
images/
  about/
  Flowers/
  index/
  Products/
  team/
script/
  about.js
  anime.js
  index.js
  login.js
  Products.js
  shopping.js
```

## 本地运行

项目是静态网站，无需安装依赖。直接用浏览器打开 `html/index.html` 即可浏览，或者使用 VS Code 的 Live Server 扩展启动本地预览。

## 功能实现说明

- `script/index.js` 实现首页视差滚动效果
- `script/Products.js` 负责产品分类渲染、评分展示和详情弹窗
- `script/shopping.js` 负责购物车数据、本地存储和金额计算
- `script/about.js` 负责数据递增动画、视差与卡片悬停效果
- `script/login.js` 负责登录注册表单切换和基础校验

## 备注

- 站点中的部分图片资源需要确保与 HTML 中引用的路径一致
- 项目当前以静态展示为主，购物、登录等流程未接入后端服务

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
