package ginswagger

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WrapHandler wraps swagger files handler
func WrapHandler(args ...interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("any")
		// 移除开头的斜杠（如果有）
		path = strings.TrimPrefix(path, "/")

		// 规范化路径：移除尾部斜杠
		path = strings.TrimSuffix(path, "/")

		// 处理 swagger.json/doc.json 请求
		if path == "doc.json" || strings.HasSuffix(path, "/doc.json") || strings.Contains(path, "doc.json") {
			swaggerJSON := generateSwaggerJSON()
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.String(http.StatusOK, swaggerJSON)
			return
		}

		// 处理 index.html 请求或空路径
		if path == "index.html" || path == "" {
			html := generateSwaggerHTML()
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.String(http.StatusOK, html)
			return
		}

		// 对于其他路径，如果是静态资源请求（包含文件扩展名），返回404
		if strings.Contains(path, ".") && !strings.HasSuffix(path, ".html") {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "path": path})
			return
		}

		// 其他情况（包括 /swagger/ 等）返回 index.html（SPA 路由）
		html := generateSwaggerHTML()
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.String(http.StatusOK, html)
	}
}

func generateSwaggerJSON() string {
	// 说明：
	// 1）此模块为本项目内的 gin-swagger 本地 stub，实现最小可用的 Swagger JSON，
	//    以保证 /swagger/index.html 能正常加载而不依赖外部模块或循环引用本项目；
	// 2）真实接口明细目前在前端 `ApiDocsView.vue` 内进行了详细呈现，
	//    该 JSON 主要用于让 Swagger UI 正常渲染基础信息与安全定义。
	return `{
  "swagger": "2.0",
  "info": {
    "description": "CYP-Registry 容器镜像仓库管理系统 RESTful 接口文档",
    "title": "CYP-Registry 容器镜像仓库管理 API",
    "version": "1.0.1",
    "contact": {
      "name": "CYP-Registry 技术支持",
      "email": "nasDSSCYP@outlook.com"
    },
    "license": {
      "name": "MIT",
      "url": "https://opensource.org/licenses/MIT"
    }
  },
  "host": "",
  "basePath": "/api/v1",
  "schemes": ["http", "https"],
  "paths": {},
  "securityDefinitions": {
    "访问令牌": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header"
    }
  },
  "security": [
    {
      "访问令牌": []
    }
  ]
}`
}

func generateSwaggerHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Swagger UI - CYP-Registry API</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
  <style>
    html {
      box-sizing: border-box;
      overflow: -moz-scrollbars-vertical;
      overflow-y: scroll;
    }
    *, *:before, *:after {
      box-sizing: inherit;
    }
    body {
      margin: 0;
      background: #fafafa;
    }
    .swagger-ui .topbar {
      display: none;
    }
    #loading {
      text-align: center;
      padding: 50px;
      font-family: Arial, sans-serif;
      color: #333;
    }
  </style>
</head>
<body>
  <div id="swagger-ui">
    <div id="loading">正在加载 Swagger UI...</div>
  </div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-standalone-preset.js"></script>
  <script>
    function localizeSwaggerUIText() {
      try {
        // 只处理文本节点，不破坏 HTML 结构
        function replaceTextInNode(node, oldText, newText) {
          if (node.nodeType === Node.TEXT_NODE) {
            var text = node.textContent;
            if (text.includes(oldText)) {
              node.textContent = text.replace(new RegExp(oldText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi'), newText);
              return true;
            }
          }
          return false;
        }

        // 使用 TreeWalker 遍历所有文本节点并替换（保持结构）
        var walker = document.createTreeWalker(
          document.body,
          NodeFilter.SHOW_TEXT,
          {
            acceptNode: function(node) {
              // 跳过 script 和 style 标签内的文本
              var parent = node.parentElement;
              if (parent && (parent.tagName === 'SCRIPT' || parent.tagName === 'STYLE')) {
                return NodeFilter.FILTER_REJECT;
              }
              return NodeFilter.FILTER_ACCEPT;
            }
          },
          false
        );

        var textNode;
        while (textNode = walker.nextNode()) {
          var text = textNode.textContent;
          // 只替换完整的单词/短语，避免部分替换
          if (text.trim() === 'Available authorizations' || text.includes('Available authorizations')) {
            textNode.textContent = text.replace(/Available authorizations/gi, '可用授权方式');
          } else if (text.trim() === 'Bearer (apiKey)' || text.match(/Bearer\s*\(apiKey\)/i)) {
            // 兼容旧 scheme 名称
            textNode.textContent = text.replace(/Bearer\s*\(apiKey\)/gi, '访问令牌（API密钥）');
          } else if (text.trim() === 'Name: Authorization' || text.match(/Name:\s*Authorization/i)) {
            textNode.textContent = text.replace(/Name:\s*Authorization/gi, '名称: Authorization');
          } else if (text.trim() === 'In: header' || text.match(/In:\s*header/i)) {
            textNode.textContent = text.replace(/In:\s*header/gi, '位置: 请求头');
          } else if (text.trim() === 'Value:' || text.match(/^Value:\s*$/i)) {
            textNode.textContent = text.replace(/Value:/gi, '值:');
          } else if (text.trim() === 'Authorize') {
            textNode.textContent = text.replace(/Authorize/gi, '授权');
          } else if (text.trim() === 'Close') {
            textNode.textContent = text.replace(/Close/gi, '关闭');
          } else if (text.trim() === 'header' && !text.includes('请求头')) {
            textNode.textContent = text.replace(/header/gi, '请求头');
          }
        }

        // 处理授权对话框中的特定元素（只处理没有子元素的元素）
        var authContainer = document.querySelector('.auth-container, .dialog-ux, .modal-ux');
        if (authContainer) {
          // 授权对话框标题 - 只处理标题元素本身，不处理子元素
          var authTitle = authContainer.querySelector('h3, h4, .modal-title, [class*="title"]');
          if (authTitle) {
            // 只处理直接文本内容，不处理子元素
            var titleWalker = document.createTreeWalker(
              authTitle,
              NodeFilter.SHOW_TEXT,
              null,
              false
            );
            var titleTextNode;
            while (titleTextNode = titleWalker.nextNode()) {
              var titleText = titleTextNode.textContent;
              if (titleText.toLowerCase().includes('available authorizations') || 
                  titleText.toLowerCase().includes('authorizations')) {
                titleTextNode.textContent = titleText.replace(/Available authorizations/gi, '可用授权方式');
              }
            }
          }

          // 处理按钮 - 只替换按钮内的文本节点
          var authorizeBtns = authContainer.querySelectorAll('button, .btn, [class*="authorize"]');
          authorizeBtns.forEach(function(btn) {
            var btnWalker = document.createTreeWalker(
              btn,
              NodeFilter.SHOW_TEXT,
              null,
              false
            );
            var btnTextNode;
            while (btnTextNode = btnWalker.nextNode()) {
              var btnText = btnTextNode.textContent.trim();
              if (btnText.toLowerCase() === 'authorize' || btnText === 'Authorize') {
                btnTextNode.textContent = btnTextNode.textContent.replace(/Authorize/gi, '授权');
              }
            }
          });

          var closeBtns = authContainer.querySelectorAll('button, .btn, [class*="close"], .close');
          closeBtns.forEach(function(btn) {
            var btnWalker = document.createTreeWalker(
              btn,
              NodeFilter.SHOW_TEXT,
              null,
              false
            );
            var btnTextNode;
            while (btnTextNode = btnWalker.nextNode()) {
              var btnText = btnTextNode.textContent.trim();
              if (btnText.toLowerCase() === 'close' || btnText === 'Close' || btnText === '×') {
                btnTextNode.textContent = btnTextNode.textContent.replace(/Close|×/g, '关闭');
              }
            }
          });
        }

        // 处理右上角授权按钮
        var topAuthBtn = document.querySelector('.auth-wrapper .authorize, .btn.authorize, [class*="authorize"]');
        if (topAuthBtn) {
          var topBtnWalker = document.createTreeWalker(
            topAuthBtn,
            NodeFilter.SHOW_TEXT,
            null,
            false
          );
          var topBtnTextNode;
          while (topBtnTextNode = topBtnWalker.nextNode()) {
            var topBtnText = topBtnTextNode.textContent.trim();
            if (topBtnText.toLowerCase() === 'authorize' || topBtnText === 'Authorize') {
              topBtnTextNode.textContent = topBtnTextNode.textContent.replace(/Authorize/gi, '认证');
            }
          }
        }
      } catch (e) {
        console.warn('Swagger UI 本地化失败:', e);
      }
    }

    function scheduleLocalization() {
      // 多次尝试，以适配 Swagger UI 异步渲染
      var attempts = 0;
      var maxAttempts = 20; // 增加尝试次数，确保所有内容都加载完成
      var timer = setInterval(function () {
        attempts++;
        localizeSwaggerUIText();
        // 即使达到最大尝试次数，也继续定期检查（因为用户可能打开授权对话框）
        if (attempts >= maxAttempts) {
          // 降低检查频率，但继续检查
          clearInterval(timer);
          // 每 2 秒检查一次，持续检查
          setInterval(localizeSwaggerUIText, 2000);
        }
      }, 300);
      
      // 监听 DOM 变化，当授权对话框打开时立即本地化
      var observer = new MutationObserver(function(mutations) {
        var hasAuthDialog = document.querySelector('.auth-container, .dialog-ux, .modal-ux');
        if (hasAuthDialog) {
          // 延迟一点执行，确保对话框内容已渲染
          setTimeout(localizeSwaggerUIText, 100);
        }
      });
      observer.observe(document.body, {
        childList: true,
        subtree: true
      });
    }

    (function() {
      try {
        // 使用 URL 加载 swagger.json，避免 JSON 转义问题
        const ui = SwaggerUIBundle({
          url: "./doc.json",
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIStandalonePreset
          ],
          plugins: [
            SwaggerUIBundle.plugins.DownloadUrl
          ],
          layout: "StandaloneLayout",
          validatorUrl: null,
          tryItOutEnabled: true,
          requestInterceptor: function(request) {
            // 确保请求使用正确的 base URL
            return request;
          },
          onComplete: function() {
            // 移除加载提示
            const loading = document.getElementById('loading');
            if (loading) {
              loading.remove();
            }
            // 本地化常用界面文案
            scheduleLocalization();
          },
          onFailure: function(data) {
            console.error('Swagger UI 加载失败:', data);
            document.getElementById('swagger-ui').innerHTML = 
              '<div style="padding: 20px; color: red; font-family: Arial, sans-serif; background: #fff; border: 1px solid #ddd; margin: 20px; border-radius: 4px;">' +
              '<h2>Swagger 文档加载失败</h2>' +
              '<p>无法加载 API 文档。请检查服务器是否正常运行。</p>' +
              '<p>错误信息: ' + (data.message || '未知错误') + '</p>' +
              '</div>';
          }
        });
      } catch (e) {
        console.error('Swagger UI 初始化失败:', e);
        document.getElementById('swagger-ui').innerHTML = 
          '<div style="padding: 20px; color: red; font-family: Arial, sans-serif; background: #fff; border: 1px solid #ddd; margin: 20px; border-radius: 4px;">' +
          '<h2>Swagger UI 加载失败</h2>' +
          '<p>错误信息: ' + e.message + '</p>' +
          '<p>请检查浏览器控制台（F12）获取更多信息。</p>' +
          '<p>如果问题持续，请检查网络连接或联系管理员。</p>' +
          '</div>';
      }
    })();
  </script>
</body>
</html>`
}
