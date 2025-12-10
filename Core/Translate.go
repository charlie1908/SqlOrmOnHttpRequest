package Core

//go get github.com/nicksnyder/go-i18n/v2/goi18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	shared "httpRequestName/Shared"
	"sync"
)

//go:embed i18N/*.json
var localeFS embed.FS

var (
	bundle *i18n.Bundle
	once   sync.Once
)

func initBundle() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load languages
	_, _ = bundle.LoadMessageFileFS(localeFS, "i18N/en.json")
	_, _ = bundle.LoadMessageFileFS(localeFS, "i18N/tr.json")
}

// InitTranslator initializes the i18n bundle once
func InitTranslator() {
	once.Do(func() {
		initBundle()
	})
}

// Translate returns translated message
func Translate(key string, data map[string]interface{}) string {
	// Güvenlik önlemi: bundle henüz initialize edilmemişse, initialize et
	if bundle == nil {
		InitTranslator()
	}

	lang := shared.Config.LANG
	loc := i18n.NewLocalizer(bundle, lang)
	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
	if err != nil {
		return key // çeviri bulunamazsa, key'i döndür
	}
	return msg
}

// ValidationError generates a translated validation error
func ValidationError(field, errorKey string) error {
	return fmt.Errorf(Translate(errorKey, map[string]interface{}{
		"Field": field,
	}))
}
