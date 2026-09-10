package models

// SiteConfig описывает целевой сайт для проверки.
// Структура совместима с форматами Sherlock/Maigret (поля checkType, absenceStrs, presenceStrs и т.д.)
// и расширена поддержкой регулярных выражений, редиректов, заголовков и WAF.
type SiteConfig struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	URLProbe string `yaml:"url_probe,omitempty" json:"url_probe,omitempty"`
	URLMain  string `yaml:"url_main,omitempty" json:"url_main,omitempty"`
	Engine   string `yaml:"engine,omitempty" json:"engine,omitempty"`

	RequestMethod   string            `yaml:"request_method,omitempty" json:"request_method,omitempty"`
	RequestPayload  string            `yaml:"request_payload,omitempty" json:"request_payload,omitempty"`
	RequestHeadOnly bool              `yaml:"request_head_only,omitempty" json:"request_head_only,omitempty"`
	Headers         map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// CheckType определяет стратегию проверки:
	// status_code | message | response_url | header
	CheckType string `yaml:"check_type,omitempty" json:"check_type,omitempty"`

	// --- Индикаторы ОТСУТСТВИЯ профиля ---
	AbsenceStrs    []string `yaml:"absence_strs,omitempty" json:"absence_strs,omitempty"`
	AbsenceRegexes []string `yaml:"absence_regexes,omitempty" json:"absence_regexes,omitempty"`
	ErrorCode      int      `yaml:"error_code,omitempty" json:"error_code,omitempty"`
	ErrorURL       string   `yaml:"error_url,omitempty" json:"error_url,omitempty"`

	// --- Индикаторы ПРИСУТСТВИЯ профиля ---
	PresenceStrs    []string `yaml:"presence_strs,omitempty" json:"presence_strs,omitempty"`
	PresenceRegexes []string `yaml:"presence_regexes,omitempty" json:"presence_regexes,omitempty"`

	// --- Правила редиректов ---
	FollowRedirects         bool     `yaml:"follow_redirects,omitempty" json:"follow_redirects,omitempty"`
	RedirectFailurePatterns []string `yaml:"redirect_failure_patterns,omitempty" json:"redirect_failure_patterns,omitempty"`

	// --- Проверка заголовков ---
	RequiredHeaders map[string]string `yaml:"required_headers,omitempty" json:"required_headers,omitempty"`

	// --- WAF / Защита ---
	WAFIndicators WAFConfig `yaml:"waf_indicators,omitempty" json:"waf_indicators,omitempty"`
	Protection    []string  `yaml:"protection,omitempty" json:"protection,omitempty"`
	Disabled      bool      `yaml:"disabled,omitempty" json:"disabled,omitempty"`

	Weight int      `yaml:"weight,omitempty" json:"weight,omitempty"`
	Tags   []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Валидация никнейма перед запросом
	RegexCheck string `yaml:"regex_check,omitempty" json:"regex_check,omitempty"`

	// Мета-информация (не используется при проверке, но сохраняется при конвертации)
	UsernameClaimed   string `yaml:"username_claimed,omitempty" json:"username_claimed,omitempty"`
	UsernameUnclaimed string `yaml:"username_unclaimed,omitempty" json:"username_unclaimed,omitempty"`
	AlexaRank         int    `yaml:"alexa_rank,omitempty" json:"alexa_rank,omitempty"`
	Source            string `yaml:"source,omitempty" json:"source,omitempty"`
}

// WAFConfig содержит пользовательские правила распознавания защиты.
type WAFConfig struct {
	StatusCodes    []int    `yaml:"status_codes,omitempty" json:"status_codes,omitempty"`
	HeaderKeys     []string `yaml:"header_keys,omitempty" json:"header_keys,omitempty"`
	BodySubstrings []string `yaml:"body_substrings,omitempty" json:"body_substrings,omitempty"`
}
