package service

// FilterSupportedModelScopesForPlatform keeps Antigravity-only model scope
// metadata from leaking into other platform contracts.
func FilterSupportedModelScopesForPlatform(platform string, scopes []string) []string {
	if platform != PlatformAntigravity || len(scopes) == 0 {
		return []string{}
	}
	return append([]string(nil), scopes...)
}

func sanitizeGroupSupportedModelScopes(g Group) Group {
	if len(g.SupportedModelScopes) == 0 {
		return g
	}
	g.SupportedModelScopes = FilterSupportedModelScopesForPlatform(g.Platform, g.SupportedModelScopes)
	return g
}

func sanitizeGroupsSupportedModelScopes(groups []Group) []Group {
	if len(groups) == 0 {
		return groups
	}
	out := make([]Group, 0, len(groups))
	for _, g := range groups {
		out = append(out, sanitizeGroupSupportedModelScopes(g))
	}
	return out
}
