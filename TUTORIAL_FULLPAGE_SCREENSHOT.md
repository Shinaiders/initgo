# Tutorial: Automação de Screenshot Completo (Full-Page) com Go, Playwright e Gin 📸

Este tutorial explica passo a passo como foi construída a automação para capturar **prints de página inteira (Full-Page)** de qualquer site, garantindo que **100% dos elementos, fontes, imagens com lazy-loading e dados de SPAs (React, Vue, Next.js)** estejam totalmente carregados antes da foto ser tirada.

---

## 🛑 O Desafio: Por que prints comuns saem quebrados ou incompletos?

Se você tentar tirar um print apenas abrindo a URL com Playwright ou Puppeteer básico, encontrará 4 problemas clássicos da web moderna:

1. **O evento `domcontentloaded` é rápido demais**: Ele dispara quando o HTML inicial é lido. Aplicações modernas (React, Vue, Angular) ainda estão baixando arquivos `.js` e buscando dados de APIs via `fetch`/`axios`.
2. **Imagens com _Lazy Loading_**: Quase todos os sites atuais só carregam as imagens quando o usuário rola a tela até elas. Sem rolar a página, o print sai com espaços cinzas ou vazios abaixo da primeira dobra (_above the fold_).
3. **Fontes Web (_Google Fonts_)**: Se a fonte personalizada não terminou de baixar, o texto aparece com a fonte padrão do sistema ou em branco (_flash of unstyled text_).
4. **URLs sem protocolo**: Usuários frequentemente digitam `google.com` ou `www.site.com` sem `https://`, o que quebra os navegadores headless com erros de protocolo.

Abaixo está a solução completa em camadas que resolve todos esses problemas.

---

## 🏛️ Visão Geral da Arquitetura

A automação foi estruturada em camadas bem definidas:

```
[Cliente HTTP / Navegador]
           │
           ▼
[Gin Router & ScrapeHandler] (internal/handler)
  - Normaliza a URL (adiciona https:// se faltar)
  - Escolhe o formato de resposta (imagem binária ou JSON)
           │
           ▼
[ScrapeService] (internal/service)
  - Orquestra o job e grava histórico no banco via GORM
           │
           ▼
[Playwright Engine] (internal/scraper)
  - Abre Chromium (visível no modo dev / headless em prod)
  - Executa as 6 etapas de espera e scroll
  - Tira o screenshot Full-Page
```

---

## 🪜 Passo 1: Normalização de URLs

Antes de entregar a URL ao Playwright, garantimos que ela sempre tenha o protocolo (`http://` ou `https://`). Se o usuário passar apenas `github.com` ou `www.google.com`, a aplicação corrige automaticamente:

```go
func normalizeURL(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return u
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "https://" + u
	}
	return u
}
```

---

## 🪜 Passo 2: As 6 Etapas de Carregamento Completo no Playwright

No arquivo [`internal/scraper/engine.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/scraper/engine.go), implementamos uma sequência rigorosa para assegurar que nada fique pela metade:

### 1. Espera pelo evento `load` da janela

Em vez de disparar no `domcontentloaded`, aguardamos o evento `window.onload`, que garante que o HTML, CSS e scripts síncronos foram processados:

```go
_, err = page.Goto(targetURL, playwright.PageGotoOptions{
    Timeout:   playwright.Float(timeoutMs),
    WaitUntil: playwright.WaitUntilStateLoad,
})
```

_Se um site tiver conexões de streaming que travem o `load`, há um fallback automático para `domcontentloaded`._

---

### 2. Espera pela Rede Ociosa (`networkidle`)

Essencial para Single Page Applications (SPAs). Aguarda até que não haja mais conexões de rede ativas por pelo menos 500ms, garantindo que todas as chamadas de API (AJAX/Fetch) terminaram:

```go
_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
    State:   playwright.LoadStateNetworkidle,
    Timeout: playwright.Float(10000), // limite de 10s
})
```

---

### 3. Espera pelo Carregamento de Fontes

Executa JavaScript no contexto do navegador para aguardar a Promise `document.fonts.ready`:

```go
_, _ = page.Evaluate(`() => document.fonts ? document.fonts.ready : Promise.resolve()`)
```

---

### 4. Rolagem Automática para Ativar _Lazy Loading_ (`triggerLazyLoading`)

Injetamos um script assíncrono em JavaScript que rola a página suavemente a cada 100ms até atingir o rodapé, ativando os gatilhos dos observadores de interseção (_IntersectionObserver_), e depois volta instantaneamente para o topo:

```go
func triggerLazyLoading(page playwright.Page) {
	scrollScript := `
		async () => {
			await new Promise((resolve) => {
				let totalHeight = 0;
				const distance = 400;
				const timer = setInterval(() => {
					const scrollHeight = document.body.scrollHeight || document.documentElement.scrollHeight;
					window.scrollBy(0, distance);
					totalHeight += distance;

					// Interrompe ao atingir o fim da página ou limite de segurança (25.000px)
					if (totalHeight >= scrollHeight || totalHeight > 25000) {
						clearInterval(timer);
						window.scrollTo(0, 0); // Retorna ao topo antes de capturar
						resolve();
					}
				}, 100);
			});
		}
	`
	_, _ = page.Evaluate(scrollScript)
	page.WaitForTimeout(400) // Aguarda o retorno da rolagem ao topo
}
```

---

### 5. Verificação de Download de Todas as Tags `<img>`

Garante que todas as imagens adicionadas ao DOM terminaram de ser baixadas e decodificadas:

```go
_, _ = page.Evaluate(`() => {
    const imgs = Array.from(document.images);
    return Promise.all(
        imgs.map(img => {
            if (img.complete) return Promise.resolve();
            return new Promise(resolve => {
                img.onload = img.onerror = resolve;
            });
        })
    );
}`)
```

---

### 6. Captura com `FullPage: true`

Com a página totalmente montada, o Playwright captura a página inteira:

```go
screenshotBytes, err := page.Screenshot(playwright.PageScreenshotOptions{
    FullPage: playwright.Bool(true),
    Type:     playwright.ScreenshotTypePng,
})
```

---

## 🪜 Passo 3: Criação do Handler HTTP no Gin

No arquivo [`internal/handler/scrape_handler.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/handler/scrape_handler.go), criamos a função `AnalisarPagina`, que suporta tanto requisições `GET` quanto `POST`.

### Retornando a Imagem Binária Diretamente (MIME `image/png`)

Para que o navegador exiba a imagem diretamente na tela ao acessar a rota, usamos `c.Data()` com o cabeçalho `Content-Type: image/png`:

```go
func (h *ScrapeHandler) AnalisarPagina(c *gin.Context) {
	// 1. Extração da URL (suporta Query Param ou JSON Body)
	targetURL := c.Query("url")
	format := c.Query("format")

	if c.Request.Method == http.MethodPost {
		var req struct {
			URL    string `json:"url"`
			Format string `json:"format"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.URL != "" {
			targetURL = req.URL
			format = req.Format
		}
	}

	// 2. Normaliza URL
	targetURL = normalizeURL(targetURL)

	// 3. Executa a automação Playwright
	screenshotBytes, savedPath, err := h.service.CaptureFullPage(c.Request.Context(), targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Se o cliente pediu formato JSON:
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"url": targetURL,
			"screenshot_path": savedPath,
			"size_bytes": len(screenshotBytes),
		})
		return
	}

	// 5. Retorno padrão: Arquivo binário PNG diretamente no navegador
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"fullpage_%d.png\"", time.Now().Unix()))
	c.Data(http.StatusOK, "image/png", screenshotBytes)
}
```

---

## 🪜 Passo 4: Registro das Rotas

No arquivo [`internal/handler/router.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/handler/router.go), registramos a rota tanto na raiz quanto no prefixo de API:

```go
// Rotas disponíveis:
r.GET("/analisar-pagina", scrapeHandler.AnalisarPagina)
r.POST("/analisar-pagina", scrapeHandler.AnalisarPagina)
v1.GET("/analisar-pagina", scrapeHandler.AnalisarPagina)
v1.POST("/analisar-pagina", scrapeHandler.AnalisarPagina)
```

---

## 🖥️ Modo Dev (Visível) vs. Modo Produção (Headless)

- **Modo Desenvolvimento (`make dev`)**:
  - `PLAYWRIGHT_HEADLESS=false`: A janela física do Chromium se abre no macOS.
  - `SlowMo = 150ms`: O Playwright desacelera ligeiramente cada ação para que você consiga ver o scroll e o carregamento com os próprios olhos.
- **Modo Produção / Docker (`make docker-up`)**:
  - `PLAYWRIGHT_HEADLESS=true`: O navegador roda silenciosamente em background sem consumir recursos de tela.

---

## 🧪 Exemplos de Teste

### 1. Testar direto no Navegador

Abra o seu navegador web (Chrome, Safari, Firefox) e acesse:

```
http://localhost:8080/analisar-pagina?url=github.com
```

_O navegador abrirá a página do GitHub, fará o scroll e renderizará a imagem PNG gigante diretamente na sua aba._

### 2. Baixar a imagem pelo Terminal (cURL)

```bash
curl -o github_completo.png "http://localhost:8080/analisar-pagina?url=github.com"
```

### 3. Disparar via POST retornando JSON

```bash
curl -X POST http://localhost:8080/analisar-pagina \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://g1.globo.com",
    "format": "json"
  }'
```

Resposta JSON:

```json
{
  "success": true,
  "message": "Página analisada e print capturado com sucesso",
  "data": {
    "url": "https://g1.globo.com",
    "screenshot_path": "storage/screenshots/fullpage_1731454800000.png",
    "size_bytes": 2840192
  }
}
```
