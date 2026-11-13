package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"goscrapper/internal/config"
	"goscrapper/internal/model"
	"goscrapper/internal/storage"

	"github.com/mxschmitt/playwright-go"
)

type Engine struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	cfg     *config.Config
	storage storage.StorageProvider
	mu      sync.Mutex
}

func NewEngine(cfg *config.Config, store storage.StorageProvider) (*Engine, error) {
	log.Println("[PLAYWRIGHT] Inicializando runtime do Playwright...")
	pw, err := playwright.Run()
	if err != nil {
		log.Printf("[PLAYWRIGHT] Driver não encontrado ou não iniciado (%v). Tentando auto-instalação...", err)
		if installErr := playwright.Install(); installErr != nil {
			return nil, fmt.Errorf("falha na auto-instalação do Playwright: %w (dica: execute 'make install-playwright')", installErr)
		}
		pw, err = playwright.Run()
		if err != nil {
			return nil, fmt.Errorf("falha ao inicializar Playwright após instalação: %w", err)
		}
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.BrowserHeadless),
	}

	if cfg.BrowserHeadless {
		// Modo Headless (ideal para Docker e produção)
		launchOptions.Args = []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
		}
	} else {
		// Modo Headed (Desenvolvimento): abre a janela visual do navegador e desacelera levemente para visualização
		launchOptions.SlowMo = playwright.Float(150)
	}

	log.Printf("[PLAYWRIGHT] Iniciando Chromium (Headless: %v)...", cfg.BrowserHeadless)
	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		_ = pw.Stop()
		return nil, fmt.Errorf("falha ao iniciar browser Chromium: %w", err)
	}

	log.Println("[PLAYWRIGHT] Browser Chromium iniciado com sucesso.")
	return &Engine{
		pw:      pw,
		browser: browser,
		cfg:     cfg,
		storage: store,
	}, nil
}

// normalizeURL garante que a URL contenha o protocolo http:// ou https://
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

// Scrape executa uma sessão de scraping isolada em um BrowserContext próprio
func (e *Engine) Scrape(ctx context.Context, req *model.ScrapeRequest) (*model.ScrapeResult, error) {
	startTime := time.Now()
	req.URL = normalizeURL(req.URL)

	// BrowserContext garante isolamento de cookies, storage e cache por requisição
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	bContext, err := e.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(userAgent),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar contexto do browser: %w", err)
	}
	defer bContext.Close()

	page, err := bContext.NewPage()
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir nova página: %w", err)
	}
	defer page.Close()

	timeoutMs := float64(e.cfg.BrowserTimeout * 1000)

	// Navega até a URL solicitada
	response, err := page.Goto(req.URL, playwright.PageGotoOptions{
		Timeout:   playwright.Float(timeoutMs),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		// Fallback para domcontentloaded caso o evento load demore
		response, err = page.Goto(req.URL, playwright.PageGotoOptions{
			Timeout:   playwright.Float(timeoutMs),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err != nil {
			return nil, fmt.Errorf("erro ao carregar URL (%s): %w", req.URL, err)
		}
	}

	// Aguarda rede acalmar se possível
	_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(8000),
	})

	statusCode := 200
	if response != nil {
		statusCode = response.Status()
	}

	// Se um seletor específico foi informado para esperar (útil para SPAs em React/Vue)
	if req.WaitSelector != "" {
		_, err = page.WaitForSelector(req.WaitSelector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(timeoutMs),
		})
		if err != nil {
			log.Printf("[WARN] Timeout esperando seletor %s: %v", req.WaitSelector, err)
		}
	}

	// Extrai título da página
	title, _ := page.Title()

	// Extrai texto visível da página
	bodyText, err := page.InnerText("body")
	if err != nil {
		bodyText = ""
	}

	// Limita texto para não sobrecarregar banco se a página for gigantesca (ex: 20000 caracteres)
	if len(bodyText) > 20000 {
		bodyText = bodyText[:20000] + "... [truncado]"
	}

	var screenshotPath string
	var screenshotBytes []byte

	// Tira screenshot se solicitado
	if req.TakeScreenshot {
		bytes, err := page.Screenshot(playwright.PageScreenshotOptions{
			FullPage: playwright.Bool(false),
		})
		if err != nil {
			log.Printf("[WARN] Falha ao capturar screenshot: %v", err)
		} else {
			screenshotBytes = bytes
			if e.storage != nil {
				filename := fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano())
				savedLocation, saveErr := e.storage.Save(ctx, filename, bytes, "image/png")
				if saveErr != nil {
					log.Printf("[WARN] Falha ao salvar screenshot no provedor de storage: %v", saveErr)
				} else {
					screenshotPath = savedLocation
				}
			}
		}
	}

	executionTime := time.Since(startTime).Milliseconds()

	return &model.ScrapeResult{
		PageTitle:       title,
		HTTPStatusCode:  statusCode,
		ContentText:     bodyText,
		ScreenshotBytes: screenshotBytes,
		ScreenshotPath:  screenshotPath,
		ExecutionTimeMs: executionTime,
	}, nil
}

// CaptureFullPageScreenshot navega até a URL e captura o screenshot completo (fullpage) da página
func (e *Engine) CaptureFullPageScreenshot(ctx context.Context, targetURL string) ([]byte, string, error) {
	targetURL = normalizeURL(targetURL)
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	bContext, err := e.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(userAgent),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("erro ao criar contexto do browser: %w", err)
	}
	defer bContext.Close()

	page, err := bContext.NewPage()
	if err != nil {
		return nil, "", fmt.Errorf("erro ao abrir página: %w", err)
	}
	defer page.Close()

	timeoutMs := float64(e.cfg.BrowserTimeout * 1000)

	log.Printf("[PLAYWRIGHT] Carregando página completamente: %s", targetURL)

	// 1. Navega aguardando o evento de 'load' completo da página (HTML, CSS, imagens e scripts)
	_, err = page.Goto(targetURL, playwright.PageGotoOptions{
		Timeout:   playwright.Float(timeoutMs),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		// Fallback para Domcontentloaded se a página demorar muito no evento 'load'
		log.Printf("[PLAYWRIGHT] Timeout no evento 'load' para %s (%v). Tentando com 'domcontentloaded'...", targetURL, err)
		_, err = page.Goto(targetURL, playwright.PageGotoOptions{
			Timeout:   playwright.Float(timeoutMs),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if err != nil {
			return nil, "", fmt.Errorf("falha ao carregar a página (%s): %w", targetURL, err)
		}
	}

	// 2. Aguarda conexões de rede cessarem (networkidle - garante que SPAs em React/Vue terminaram de carregar APIs)
	log.Println("[PLAYWRIGHT] Aguardando rede ociosa (networkidle)...")
	_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(10000), // Tolera até 10s para a rede acalmar
	})

	// 3. Aguarda o carregamento completo de fontes da página (evita textos em branco/desalinhados)
	_, _ = page.Evaluate(`() => document.fonts ? document.fonts.ready : Promise.resolve()`)

	// 4. Rola a página até o fim e volta para o topo para disparar imagens e componentes 'lazy-loaded'
	log.Println("[PLAYWRIGHT] Rolando a página para disparar imagens lazy-loaded...")
	triggerLazyLoading(page)

	// 5. Garante que todas as tags <img> presentes no DOM finalizaram o download
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

	// 6. Pequena pausa de 500ms para estabilização de animações CSS
	page.WaitForTimeout(500)

	log.Println("[PLAYWRIGHT] Página 100% carregada. Capturando print inteiro (FullPage)...")

	// 7. Captura do print INTEIRO do site (FullPage: true)
	screenshotBytes, err := page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(true),
		Type:     playwright.ScreenshotTypePng,
	})
	if err != nil {
		return nil, "", fmt.Errorf("falha ao capturar screenshot inteira: %w", err)
	}

	var savedPath string
	if e.storage != nil {
		filename := fmt.Sprintf("fullpage_%d.png", time.Now().UnixNano())
		savedPath, _ = e.storage.Save(ctx, filename, screenshotBytes, "image/png")
	}

	return screenshotBytes, savedPath, nil
}

// triggerLazyLoading rola a página suavemente até o rodapé para forçar o carregamento de imagens lazy-loaded e depois retorna ao topo
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

					// Interrompe quando atingir o final da página ou limite de segurança (25.000px)
					if (totalHeight >= scrollHeight || totalHeight > 25000) {
						clearInterval(timer);
						window.scrollTo(0, 0); // Retorna ao topo antes do print
						resolve();
					}
				}, 100);
			});
		}
	`
	_, _ = page.Evaluate(scrollScript)
	page.WaitForTimeout(400) // Aguarda o scroll voltar ao topo
}

// Close finaliza o browser e o runtime do Playwright
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	log.Println("[PLAYWRIGHT] Finalizando browser...")
	if e.browser != nil {
		_ = e.browser.Close()
	}
	if e.pw != nil {
		_ = e.pw.Stop()
	}
	log.Println("[PLAYWRIGHT] Browser e runtime finalizados com sucesso.")
}
