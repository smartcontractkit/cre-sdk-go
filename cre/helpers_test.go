package cre

import "sync"

func ClearRepostSignatureCache() {
	keyCache = &sync.Map{}
}
