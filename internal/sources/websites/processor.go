package websites

import (
	"context"
	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/security"
	"github.com/johan-larp/agentsearch/internal/worker"
	"io"
	"net/http"
	"strings"
	"time"
)

// siteProcessor реализует worker.Processor — логику обработки одного сайта.
type siteProcessor struct {
	source *Source
	target string
	kind   models.TargetType
}

// Process выполняет HTTP-запрос и применяет декларативный движок детекции.
func (p *siteProcessor) Process(ctx context.Context, job worker.Job) (res models.Result) {
	site := job.Site
	start := time.Now()

	// Подстановка переменных в URL
	checkURL := strings.ReplaceAll(site.URL, "{username}", p.target)
	checkURL = strings.ReplaceAll(checkURL, "{target}", p.target)
	// urlProbe имеет приоритет, если задан
	if site.URLProbe != "" {
		checkURL = strings.ReplaceAll(site.URLProbe, "{username}", p.target)
		checkURL = strings.ReplaceAll(checkURL, "{target}", p.target)
	}

	res = models.Result{
		Source:     site.Name,
		SourceType: models.SourceWebsite,
		TargetType: p.kind,
		SiteName:   site.Name,
		Target:     p.target,
		URL:        checkURL,
		Status:     models.StatusError,
	}

	// Include elapsed time and redact configured credentials on every return path.
	secrets := make([]string, 0)
	for key, value := range site.Headers {
		if security.SensitiveKey(key) {
			secrets = append(secrets, value)
		}
	}
	defer func() { res.Duration = time.Since(start); res = res.Redacted(secrets...).Normalized() }()
	// Rate limiting per host
	if err := p.source.limiter.WaitContext(ctx, checkURL); err != nil {
		res.Error = err.Error()
		return res
	}

	// Формирование запроса
	method := http.MethodGet
	if site.RequestMethod != "" {
		method = site.RequestMethod
	}
	if site.RequestHeadOnly && method == http.MethodGet {
		method = http.MethodHead
	}

	var bodyReader io.Reader
	if site.RequestPayload != "" {
		payload := strings.ReplaceAll(site.RequestPayload, "{username}", p.target)
		payload = strings.ReplaceAll(payload, "{target}", p.target)
		bodyReader = strings.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, checkURL, bodyReader)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	// Заголовки: ротация UA + кастомные заголовки сайта
	req.Header.Set("User-Agent", p.source.ua.GetRandom())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	for k, v := range site.Headers {
		req.Header.Set(k, strings.ReplaceAll(v, "{username}", p.target))
	}

	// Выполнение запроса с контролем редиректов
	client := p.source.client
	if !site.FollowRedirects {
		noRedirect := *client
		noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		client = &noRedirect
	}

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	// Определяем финальный URL
	finalURL := checkURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	// Читаем тело с ограничением (128 KiB) — защита от огромных ответов
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	body := string(bodyBytes)

	// Декларативная детекция
	detection := p.source.detector.Analyze(site, resp, body, finalURL)
	res.Found = detection.Found
	res.Confidence = detection.Confidence
	res.Status = detection.Status
	res.Duration = time.Since(start)
	if finalURL != checkURL {
		res.FinalURL = finalURL
	}

	return res
}
