package controller

import (
	"encoding/json"
	"strings"
)

const simplifiedChineseLanguage = "zh-Hans"

func multilingualTextValue(chinese string, english string) string {
	encoded, err := json.Marshal(struct {
		Chinese string `json:"zh-Hans"`
		English string `json:"en"`
	}{
		Chinese: chinese,
		English: english,
	})
	if err != nil {
		return chinese
	}
	return string(encoded)
}

// localizedTextValue keeps legacy plain-text values unchanged. A value is
// treated as multilingual content only when it is a JSON object containing at
// least one supported language key.
func localizedTextValue(value string, language string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '{' {
		return value
	}

	var translations map[string]string
	if err := json.Unmarshal([]byte(trimmed), &translations); err != nil {
		return value
	}

	var chinese, english string
	var hasChinese, hasEnglish bool
	for key, translation := range translations {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "zh-hans":
			chinese, hasChinese = translation, true
		case "en":
			english, hasEnglish = translation, true
		}
	}
	if !hasChinese && !hasEnglish {
		return value
	}

	if language == simplifiedChineseLanguage || language == "" {
		if hasChinese && chinese != "" {
			return chinese
		}
		return english
	}
	if hasEnglish && english != "" {
		return english
	}
	return chinese
}
