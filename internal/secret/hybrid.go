package secret

// hybridStore prefers the OS keychain but transparently falls back to an
// encrypted local file on a per-secret basis, not just when no keychain
// exists at all: OS keychains also cap how large a single item's secret can
// be (a few KB), and a full ChatGPT-OAuth auth.json blob — id_token,
// access_token, refresh_token — routinely exceeds that. Trying the keychain
// first and falling back on any error covers both "no keychain" (NFR-03/
// FR-02) and "keychain present but rejected this item" uniformly.
type hybridStore struct {
	primary  Store
	fallback Store
}

func (h *hybridStore) Backend() string {
	return "OS keychain + encrypted-file fallback"
}

func (h *hybridStore) Set(profileName, value string) error {
	if err := h.primary.Set(profileName, value); err == nil {
		// Clear any stale fallback copy from a prior attempt so Get/Delete
		// don't need to reconcile two possible locations going forward.
		_ = h.fallback.Delete(profileName)
		return nil
	}
	return h.fallback.Set(profileName, value)
}

func (h *hybridStore) Get(profileName string) (string, error) {
	if v, err := h.primary.Get(profileName); err == nil {
		return v, nil
	}
	return h.fallback.Get(profileName)
}

func (h *hybridStore) Delete(profileName string) error {
	primaryErr := h.primary.Delete(profileName)
	fallbackErr := h.fallback.Delete(profileName)
	if primaryErr != nil {
		return primaryErr
	}
	return fallbackErr
}
