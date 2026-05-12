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

	return &mcommon.UserPrivacy{
		IsConfigured:       true,
		LogRequestContent:  logPrivacy.IsEnableRequest && privacy.LogRequestContent,
		LogResponseContent: logPrivacy.IsEnableResponse && privacy.LogResponseContent,
		LogResourceUrl:     logPrivacy.IsEnableResource && privacy.LogResourceUrl,
		LogClientIp:        logPrivacy.IsEnableNetwork && privacy.LogClientIp,
		LogRequestFields:   normalizePrivacyFields(privacy.LogRequestFields, logPrivacy.RequestPrivacyFields, logPrivacy.IsEnableRequest && privacy.LogRequestContent),
		LogResponseFields:  normalizePrivacyFields(privacy.LogResponseFields, logPrivacy.ResponsePrivacyFields, logPrivacy.IsEnableResponse && privacy.LogResponseContent),
		LogResourceFields:  normalizePrivacyFields(privacy.LogResourceFields, logPrivacy.ResourcePrivacyFields, logPrivacy.IsEnableResource && privacy.LogResourceUrl),
		LogNetworkFields:   normalizePrivacyFields(privacy.LogNetworkFields, logPrivacy.NetworkPrivacyFields, logPrivacy.IsEnableNetwork && privacy.LogClientIp),
	}
}

func DefaultLogUserPrivacy(logPrivacy *mcommon.LogPrivacy) *mcommon.UserPrivacy {

	if logPrivacy == nil {
		return DefaultUserPrivacy()
	}

	return &mcommon.UserPrivacy{
		LogRequestContent:  logPrivacy.IsEnableRequest && logPrivacy.IsDefaultEnableRequest,
		LogResponseContent: logPrivacy.IsEnableResponse && logPrivacy.IsDefaultEnableResponse,
		LogResourceUrl:     logPrivacy.IsEnableResource && logPrivacy.IsDefaultEnableResource,
		LogClientIp:        logPrivacy.IsEnableNetwork && logPrivacy.IsDefaultEnableNetwork,
		LogRequestFields:   defaultPrivacyFields(logPrivacy.RequestPrivacyFields, logPrivacy.IsEnableRequest && logPrivacy.IsDefaultEnableRequest),
		LogResponseFields:  defaultPrivacyFields(logPrivacy.ResponsePrivacyFields, logPrivacy.IsEnableResponse && logPrivacy.IsDefaultEnableResponse),
		LogResourceFields:  defaultPrivacyFields(logPrivacy.ResourcePrivacyFields, logPrivacy.IsEnableResource && logPrivacy.IsDefaultEnableResource),
		LogNetworkFields:   defaultPrivacyFields(logPrivacy.NetworkPrivacyFields, logPrivacy.IsEnableNetwork && logPrivacy.IsDefaultEnableNetwork),
	}
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
