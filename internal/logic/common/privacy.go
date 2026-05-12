package common

import mcommon "github.com/iimeta/fastapi/v2/internal/model/common"

func DefaultUserPrivacy() *mcommon.UserPrivacy {
	return &mcommon.UserPrivacy{
		LogRequestFields:  []string{},
		LogResponseFields: []string{},
		LogResourceFields: []string{},
		LogNetworkFields:  []string{},
	}
}

func NormalizeUserPrivacy(privacy *mcommon.UserPrivacy, logPrivacy *mcommon.LogPrivacy) *mcommon.UserPrivacy {

	if logPrivacy == nil {
		return DefaultUserPrivacy()
	}

	if privacy == nil || !privacy.IsConfigured {
		return DefaultLogUserPrivacy(logPrivacy)
	}

	logRequestContent := categoryPrivacyEnabled(true, logPrivacy.IsEnableRequest, logPrivacy.IsDefaultEnableRequest, privacy.LogRequestContent)
	logResponseContent := categoryPrivacyEnabled(true, logPrivacy.IsEnableResponse, logPrivacy.IsDefaultEnableResponse, privacy.LogResponseContent)
	logResourceUrl := categoryPrivacyEnabled(true, logPrivacy.IsEnableResource, logPrivacy.IsDefaultEnableResource, privacy.LogResourceUrl)
	logClientIp := categoryPrivacyEnabled(true, logPrivacy.IsEnableNetwork, logPrivacy.IsDefaultEnableNetwork, privacy.LogClientIp)

	return &mcommon.UserPrivacy{
		IsConfigured:       true,
		LogRequestContent:  logRequestContent,
		LogResponseContent: logResponseContent,
		LogResourceUrl:     logResourceUrl,
		LogClientIp:        logClientIp,
		LogRequestFields:   categoryPrivacyFields(privacy.LogRequestFields, logPrivacy.RequestPrivacyFields, logPrivacy.IsEnableRequest, logRequestContent),
		LogResponseFields:  categoryPrivacyFields(privacy.LogResponseFields, logPrivacy.ResponsePrivacyFields, logPrivacy.IsEnableResponse, logResponseContent),
		LogResourceFields:  categoryPrivacyFields(privacy.LogResourceFields, logPrivacy.ResourcePrivacyFields, logPrivacy.IsEnableResource, logResourceUrl),
		LogNetworkFields:   categoryPrivacyFields(privacy.LogNetworkFields, logPrivacy.NetworkPrivacyFields, logPrivacy.IsEnableNetwork, logClientIp),
	}
}

func DefaultLogUserPrivacy(logPrivacy *mcommon.LogPrivacy) *mcommon.UserPrivacy {

	if logPrivacy == nil {
		return DefaultUserPrivacy()
	}

	return &mcommon.UserPrivacy{
		LogRequestContent:  logPrivacy.IsDefaultEnableRequest,
		LogResponseContent: logPrivacy.IsDefaultEnableResponse,
		LogResourceUrl:     logPrivacy.IsDefaultEnableResource,
		LogClientIp:        logPrivacy.IsDefaultEnableNetwork,
		LogRequestFields:   defaultPrivacyFields(logPrivacy.RequestPrivacyFields, logPrivacy.IsDefaultEnableRequest),
		LogResponseFields:  defaultPrivacyFields(logPrivacy.ResponsePrivacyFields, logPrivacy.IsDefaultEnableResponse),
		LogResourceFields:  defaultPrivacyFields(logPrivacy.ResourcePrivacyFields, logPrivacy.IsDefaultEnableResource),
		LogNetworkFields:   defaultPrivacyFields(logPrivacy.NetworkPrivacyFields, logPrivacy.IsDefaultEnableNetwork),
	}
}

func categoryPrivacyEnabled(configured bool, allowUserConfig bool, defaultEnabled bool, userEnabled bool) bool {

	if !configured || !allowUserConfig {
		return defaultEnabled
	}

	return userEnabled
}

func categoryPrivacyFields(values []string, fields []mcommon.PrivacyLogFieldOption, allowUserConfig bool, enabled bool) []string {

	if !enabled {
		return []string{}
	}

	if !allowUserConfig {
		return defaultPrivacyFields(fields, true)
	}

	return normalizePrivacyFields(values, fields, true)
}

func enabledPrivacyLogFields(fields []mcommon.PrivacyLogFieldOption) []mcommon.PrivacyLogFieldOption {

	items := make([]mcommon.PrivacyLogFieldOption, 0)
	for _, field := range fields {
		if field.Enabled && field.Key != "" {
			items = append(items, field)
		}
	}

	return items
}

func defaultPrivacyFields(fields []mcommon.PrivacyLogFieldOption, enabled bool) []string {

	if !enabled {
		return []string{}
	}

	result := make([]string, 0)
	for _, field := range enabledPrivacyLogFields(fields) {
		result = append(result, field.Key)
	}

	return result
}

func normalizePrivacyFields(values []string, fields []mcommon.PrivacyLogFieldOption, enabled bool) []string {

	if !enabled {
		return []string{}
	}

	allowed := map[string]bool{}
	for _, field := range enabledPrivacyLogFields(fields) {
		allowed[field.Key] = true
	}

	result := make([]string, 0)
	seen := map[string]bool{}

	for _, value := range values {
		if allowed[value] && !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}

	return result
}
