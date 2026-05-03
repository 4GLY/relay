package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	relayLocaleCookie = "relay_locale"

	publicSnapshotLocaleEN = "en"
	publicSnapshotLocaleKO = "ko"
)

type publicSnapshotMessages struct {
	HTMLLang           string
	SnapshotLabel      string
	TitleSuffix        string
	DefaultTaskSummary string
	OGDescription      string
	FooterText         string
	FooterCTA          string
}

var publicSnapshotMessagesByLocale = map[string]publicSnapshotMessages{
	publicSnapshotLocaleEN: {
		HTMLLang:           "en",
		SnapshotLabel:      "Snapshot",
		TitleSuffix:        "Relay snapshot",
		DefaultTaskSummary: "Snapshot from Relay",
		OGDescription:      "A public Relay snapshot with captured judgment and context.",
		FooterText:         "A Relay snapshot — captured judgment, kept honest.",
		FooterCTA:          "Make your own at relay.4gly.dev →",
	},
	publicSnapshotLocaleKO: {
		HTMLLang:           "ko",
		SnapshotLabel:      "스냅샷",
		TitleSuffix:        "Relay 스냅샷",
		DefaultTaskSummary: "Relay에서 만든 스냅샷",
		OGDescription:      "판단과 맥락을 담은 공개 Relay 스냅샷입니다.",
		FooterText:         "Relay 스냅샷 — 판단을 기록하고 맥락을 지킵니다.",
		FooterCTA:          "relay.4gly.dev에서 직접 만들기 →",
	},
}

func resolvePublicSnapshotMessages(r *http.Request) publicSnapshotMessages {
	locale := resolvePublicSnapshotLocale(r)
	messages, ok := publicSnapshotMessagesByLocale[locale]
	if !ok {
		return publicSnapshotMessagesByLocale[publicSnapshotLocaleEN]
	}
	return messages
}

func resolvePublicSnapshotLocale(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie(relayLocaleCookie); err == nil {
			if locale := normalizePublicSnapshotLocale(cookie.Value); locale != "" {
				return locale
			}
		}
		for _, candidate := range parseAcceptLanguage(r.Header.Get("Accept-Language")) {
			if locale := normalizePublicSnapshotLocale(candidate); locale != "" {
				return locale
			}
		}
	}
	return publicSnapshotLocaleEN
}

func normalizePublicSnapshotLocale(value string) string {
	candidate := strings.TrimSpace(strings.ToLower(value))
	if candidate == "ko" || strings.HasPrefix(candidate, "ko-") {
		return publicSnapshotLocaleKO
	}
	if candidate == "en" || strings.HasPrefix(candidate, "en-") {
		return publicSnapshotLocaleEN
	}
	return ""
}

func parseAcceptLanguage(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	type languagePreference struct {
		tag   string
		q     float64
		index int
	}

	parts := strings.Split(header, ",")
	preferences := make([]languagePreference, 0, len(parts))
	for index, part := range parts {
		segments := strings.Split(strings.TrimSpace(part), ";")
		tag := strings.TrimSpace(segments[0])
		if tag == "" {
			continue
		}

		q := 1.0
		for _, segment := range segments[1:] {
			param := strings.TrimSpace(segment)
			if !strings.HasPrefix(param, "q=") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(param, "q=")), 64)
			if err != nil {
				q = 0
				break
			}
			q = parsed
			break
		}
		if q <= 0 {
			continue
		}
		preferences = append(preferences, languagePreference{tag: tag, q: q, index: index})
	}

	sort.SliceStable(preferences, func(i, j int) bool {
		if preferences[i].q == preferences[j].q {
			return preferences[i].index < preferences[j].index
		}
		return preferences[i].q > preferences[j].q
	})

	locales := make([]string, 0, len(preferences))
	for _, preference := range preferences {
		locales = append(locales, preference.tag)
	}
	return locales
}
