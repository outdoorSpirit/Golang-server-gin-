package lib

import (
	"os"
	"path"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	C "github.com/spiker/spiker-server/constant"
)

var bundle *i18n.Bundle

type LanguageConfiguration struct {
	Path string
}

func LanguageBundle() *i18n.Bundle {
	return bundle
}

func SetupI18n(config *LanguageConfiguration) {
	root := os.Getenv("SERVER_ROOT")

	bundle = i18n.NewBundle(language.Japanese)
	bundle.MustLoadMessageFile(path.Join(root, config.Path, "active.ja.json"))
	bundle.MustLoadMessageFile(path.Join(root, config.Path, "active.en.json"))
}

type Localizer struct {
	bundle           *i18n.Bundle
	defaultLocalizer *i18n.Localizer
	isJapanese       bool
}

func NewLocalizer(langOrders ...string) *Localizer {
	localizer := &Localizer{
		bundle:           bundle,
		defaultLocalizer: i18n.NewLocalizer(bundle, langOrders...),
	}
	localizer.isJapanese = false
	for _, lang := range langOrders {
		t, _, _ := language.ParseAcceptLanguage(lang)
		if len(t) > 0 && t[0] == language.Japanese {
			localizer.isJapanese = true
			break
		}
	}
	return localizer
}

func (l *Localizer) Localize(messageID string, templateData interface{}) string {
	msg, err := l.defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}
	return msg
}

func (l *Localizer) LocalizeWithDefault(messageID string, templateData interface{}, defaultStr string) string {
	msg, err := l.defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return defaultStr
	}
	return msg
}

func (l *Localizer) LocalizeWithLang(lang C.Language, messageID string, templateData interface{}) string {
	customLocalizer := i18n.NewLocalizer(l.bundle, string(lang))
	msg, err := customLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}
	return msg
}

func (l *Localizer) IsJapanese() bool {
	return l.isJapanese
}
